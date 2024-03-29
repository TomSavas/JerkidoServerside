package main

import (
	"bytes"
	"github.com/globalsign/mgo"
	"github.com/pkg/errors"
	"gopkg.in/mgo.v2/bson"
	"math/rand"
	"sort"
	"time"
)

type RoomState int

const (
	// Deprecated
	WaitingForPlayers RoomState = 0
	Transition        RoomState = 1
	CountingDown_3    RoomState = 2
	CountingDown_2    RoomState = 3
	CountingDown_1    RoomState = 4
	CountingDown_0    RoomState = 5
	Play              RoomState = 6
	End               RoomState = 7
)

type Room struct {
	ID                string    `json:"id"`
	State             RoomState `json:"state"`
	PlayerIDs         []string  `json:"playerids"`
	ObserverIDs       []string  `json:"observerids"`
	OwnerID           string    `json:"ownerid"`
	CreationTime      string    `json:"creationtime"`
	PlayTimeInSeconds int       `json:"playtimeinseconds"`
}

type RoomInfo struct {
	ID                string    `json:"id"`
	State             RoomState `json:"state"`
	Players           []Player  `json:"players"`
	OwnerID           string    `json:"ownerid"`
	PlayTimeInSeconds int       `json:"playtimeinseconds"`
}

func NewRoom(ownerID string, creatorIsPlayer bool, playTimeInSeconds int) *Room {
	var players []string
	var observers []string

	if creatorIsPlayer {
		players = []string{ownerID}
	} else {
		observers = []string{ownerID}
	}

	return &Room{GenerateUniqueRoomCode(GetRooms()), End, players, observers, ownerID, time.Now().UTC().String(), playTimeInSeconds}
}

func GetExistingRoom(roomID string) *Room {
	room := new(Room)
	room.ID = roomID

	room.Read()
	return room
}

func GetRooms() *mgo.Collection {
	return GetCollection("Rooms")
}

func RemoveEmptyRooms() {
	rooms := GetRooms()

	LogInfo("RoomDB", "Periodically removing rooms with empty playerids.")
	changeInfo, err := rooms.RemoveAll(bson.M{"playerids": bson.M{"$eq": []string{}}})
	changeInfoString := "Matched: " + string(changeInfo.Matched) + " removed: " + string(changeInfo.Removed)
	if err != nil {
		Log(err, "RoomDB", "Failed periodically removing rooms with empty playerids."+changeInfoString)
	} else {
		Log(err, "RoomDB", changeInfoString)
	}

	LogInfo("RoomDB", "Periodically removing rooms that have been created two hours ago.")
	changeInfo, err = rooms.RemoveAll(bson.M{"creationtime": bson.M{"$lte": time.Now().UTC().String()}})
	changeInfoString = "Matched: " + string(changeInfo.Matched) + " removed: " + string(changeInfo.Removed)
	if err != nil {
		Log(err, "RoomDB", "Failed periodically removing rooms that have been created two hours ago."+changeInfoString)
	} else {
		Log(err, "RoomDB", changeInfoString)
	}

	// Remove rooms that have playerids that are offline
	LogInfo("RoomDB", "Periodically removing rooms that have only offline players in playerids.")
	var potentiallyOfflineRooms []Room
	err = rooms.Find(bson.M{}).Limit(10000).All(&potentiallyOfflineRooms)
	Log(err, "RoomDB", "Failed periodically removing rooms that have only offline players in playerids.")

	if err != nil {
		return
	}

	for _, room := range potentiallyOfflineRooms {
		hasAnyOnlinePlayers := false

		for _, playerID := range room.PlayerIDs {
			player, _ := GetExistingPlayer(playerID)
			hasAnyOnlinePlayers = hasAnyOnlinePlayers || player.Online
		}

		if hasAnyOnlinePlayers {
			err = rooms.Remove(bson.M{"id": room.ID})
			Log(err, "RoomDB", "Failed periodical remove of room "+room.ID+".")
		}
	}
}

// Returns a value representing whether the room was upated
func (room *Room) ReadWithStatusReport() bool {
	//Bug - copies slice pointers
	oldRoom := &Room{room.ID, room.State, room.PlayerIDs, room.ObserverIDs, room.OwnerID, room.CreationTime, room.PlayTimeInSeconds}
	room.Read()

	return !room.Equals(oldRoom)
}

func (room *Room) Read() {
	err := GetRooms().Find(bson.M{"id": room.ID}).One(room)
	Fatal(err, "Failed reading a room from the database")
}

func (room *Room) AddPlayer(playerID string) {
	playerExists := false
	for _, id := range room.PlayerIDs {
		if id == playerID {
			playerExists = true
			break
		}
	}

	if !playerExists {
		room.PlayerIDs = append(room.PlayerIDs, playerID)
		_, err := room.ChangeField("playerids", room.PlayerIDs)
		Fatal(err, "Failed to update room state")
	}
}

func (room *Room) RemovePlayer(playerID string) {
	playerIndex := -1
	for index, id := range room.PlayerIDs {
		if id == playerID {
			playerIndex = index
			break
		}
	}

	if playerIndex != -1 {
		room.PlayerIDs[playerIndex] = room.PlayerIDs[len(room.PlayerIDs)-1]
		room.PlayerIDs = room.PlayerIDs[:len(room.PlayerIDs)-1]

		_, err := room.ChangeField("playerids", room.PlayerIDs)
		Fatal(err, "Failed to update room state")
	}
}

func (room *Room) ChangeRoomState(state RoomState) {
	_, err := room.ChangeField("state", state)
	Fatal(err, "Failed to update room state")
}

func (room *Room) ChangeField(fieldName string, field interface{}) (*mgo.ChangeInfo, error) {
	return GetRooms().Upsert(bson.M{"id": room.ID}, bson.M{"$set": bson.M{fieldName: field}})
}

func (room *Room) Write() *mgo.ChangeInfo {
	changeInfo, err := GetRooms().Upsert(bson.M{"id": room.ID}, bson.M{"$set": room})
	Fatal(err, "Failed to write a room")

	return changeInfo
}

func (room *Room) Info() *RoomInfo {
	players, err := room.PlayersInRoom()
	Fatal(err, "Failed forming room info")

	return &RoomInfo{room.ID, room.State, players, room.OwnerID, room.PlayTimeInSeconds}
}

func (room *Room) Equals(otherRoom *Room) bool {
	if room.ID == otherRoom.ID &&
		room.State == otherRoom.State &&
		room.OwnerID == otherRoom.OwnerID &&
		len(room.PlayerIDs) == len(otherRoom.PlayerIDs) &&
		len(room.ObserverIDs) == len(otherRoom.ObserverIDs) &&
		room.PlayTimeInSeconds == otherRoom.PlayTimeInSeconds {
		equals := func(firstList []string, secondList []string) bool {
			sort.Strings(firstList)
			sort.Strings(secondList)

			for i := 0; i < len(firstList); i++ {
				if firstList[i] != secondList[i] {
					return false
				}
			}
			return true
		}

		return equals(room.PlayerIDs, otherRoom.PlayerIDs) && equals(room.ObserverIDs, otherRoom.ObserverIDs)
	} else {
		return false
	}
}

func (room *Room) PlayersInRoom() ([]Player, error) {
	var players []Player
	var errs error

	for i := 0; i < len(room.PlayerIDs); i++ {
		player, err := GetExistingPlayer(room.PlayerIDs[i])
		if err != nil {
			errs = errors.New(errs.Error() + err.Error())
			continue
		}
		players = append(players, *player)
	}

	return players, errs
}

func GenerateUniqueRoomCode(rooms *mgo.Collection) string {
	code := GenerateRoomCode(6)

	if (rooms.Find(bson.M{"ID": code}) == nil) {
		return GenerateUniqueRoomCode(rooms)
	}

	return code
}

func GenerateRoomCode(codeLength int) string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	var code bytes.Buffer

	for i := 0; i < codeLength; i++ {
		number := rng.Intn(10)
		code.WriteString(string(number + 48))
	}

	return code.String()
}
