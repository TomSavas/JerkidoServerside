# Backend for Jerkido

What functionality it provides:
   * Global leaderboards
   * Rooms

Global leaderboards will work on _http_ as I will only need to update or get it _rarely_. 
Http requests will be served on port 8080.

Rooms will use _websocket_ connection, as I found that websocket connection is more _time efficient_ in sending _lots_ of requests in quick succession when compared to http. 
Websocket connection will be handled on port 8081.

## TODOS
 * Code cleanup (general points):
   * Merge *websocket.Conn and isConnectionClosed channel into a single struct
   * Write nice server activity logging system
   * Get rid of unnecessary ws w/rs
   * Split util functions into separate files
   * Write tests
   * Figure out how to deal nicely with _if err != nil..._ spam (monads might fit nicely for my purposes)
 * Figure out a way to not have to pull the whole database and sort it in order to get the user's global position. Current implementation will definitely be very slow and painful for the server if more people join in. Also it will slow down as the time passes.
 * Possibly drop mongo and move to simpler sql db system.
 * MAYBE security? I'm not very concerned with it as this is only meant for a game. The game has no passwords or any sensitive data that might cause trouble when intercepted or fiddled with.

## Basic database objects
Player: `{"id": "349utlkj23rojrl324ijr23_USERNAME", "score": 0, "top_score": 100, "online": 1}`
   * id - combination of user's device UID and username
   * score - current score (only used if player is currently playing in a room)
   * top_score - top score which the user has reached in either a room or a singleplayer mode
   * online - whether the user is online in a room or not.

Room: `{"id": "GR9T3", "state": 0, "playerids": ["349utlkj23rojrl324ijr23_USERNAME", ...], "observerids": ["3oroisfjjjjj309fjjsdlkf_USERNAME", ...], "ownerid": "3oroisfjjjjj309fjjsdlkf_USERNAME"}`
   * id - randomly generated string of length 5, made up of random uppercase characters and numbers
   * state - enum representing the state of the room:
      * 0 - WaitingForPlayers - room is waiting for all players to join in. Game is not started yet
      * 1 - CountingDown_3 - game is about to begin, countdown is in state "3"
      * 2 - CountingDown_2 - game is about to begin, countdown is in state "2"
      * 3 - CountingDown_1 - game is about to begin, countdown is in state "1"
      * 4 - CountingDown_0 - game is about to begin, countdown is in state "0"
      * 5 - Play - game has begun
      * 6 - End - time has ended
   * playtime - game time in seconds
   * playerids - array of ids of players (mobile clients) who are currently in the room
   * observerids - array of ids of observers (web clients) who are currently in the room
   * ownerid - id of the room owner
 
## HTTP communication:
| Endpoint | Method | Request data | Response data | Possible reponse codes |
|----------|:------:|--------------|--------------|------------------------|
| /global/top/[AMOUNT_OF_TOP_PLAYERS] | GET | - | Array of player objects sorted by top_score | `200` `500` |
| /global/position/[PLAYER_ID] | GET | - | Player's position: `{ "position" : 1 }` | `200` `500` |
| /global/save_score | PUT | Valid Player object | - | `200` `201` `500` |

## Websocket communication:
### /room/create
Either _observer_ or a _player_ establishes a connection by sending a json object: `{"id":"349utlkj23rojrl324ijr23_USERNAME", "isplayer": 0, "playtime": 30}`. If the json object is correct, then the server generates a new room and returns: `{"id": "GR9T3"}`. 

The server waits for the user to send a new json object: `{"id": "GR9T3", "state": <room state number>}`. The server will wait until the user sets the state of the room to _play_. After that is done, if `isplayer` has been set to _0_ the server will send a json object representing a room in some arbitrary intervals: `{"id":"GR9T3", "state": 5, "players": [<player objects>]}`, otherwise it will not send any information. After the playtime has ended, the server will send one last room representing json object (regardless of what `isplayer` was set to), but now the `state` field will contain code for _End_.

### /room/observe
An _observer_ establishes a connection by sending a json object: `{"id":"349utlkj23rojrl324ijr23_USERNAME", "roomid": "GR9T3"}`. If the json object is correct, then the server adds the observer to the observer list in the room. 

The server will send a json object represeting the room every time the state of the room changes. After the room state has been set to _Play_. The server will send a json object representing a room in some arbitrary intervals: `{"id":"GR9T3", "state": 5, "players": [<player objects>]}`. After the playtime has ended, the server will send one last json object representing the room (regardless of what `isplayer` was set to), but now the `state` field will contain code for _End_.

### /room/join
A _pbserver_ establishes a connection by sending a json object: `{"id":"349utlkj23rojrl324ijr23_USERNAME", "roomid": "GR9T3"}`. If the json object is correct, then the server adds the player to the player list in the room. 

The server will send a json object represeting the room every time the state of the room changes. After the room state has been set to _Play_ the server will expect the player to send a json object representing the player: `{"id":"349utlkj23rojrl324ijr23_USERNAME", "score": <current player score>}`. After the playtime has ended, the server will send one last json object representing the room, but now the `state` field will contain code for _End_.
