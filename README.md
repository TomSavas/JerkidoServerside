# Backend for Jerkido

What functionality it provides:
   * Global leaderboards
   * Rooms

## Basic database objects
Player: `{ "id" : "349utlkj23rojrl324ijr23_USERNAME", "score": 0, "top_score": 100 }`
   * id - combination of user's device UID and username
   * score - current score (only used if player is currently playing in a room)
   * top_score - top score which the user has reached in either a room or a singleplayer mode

Room: `{ "id": "GR8M8", "state": 0, "player_ids" : [ "349utlkj23rojrl324ijr23_USERNAME", ...] }`
   * id - randomly generated string of length 5, made up of random uppercase characters and numbers
   * state - enum representing the state of the room:
      * 0 - WaitingForPlayers - room is waiting for all players to join in. Game is not started yet
      * 1 - CountingDown_3 - game is about to begin, countdown is in state "3"
      * 2 - CountingDown_2 - game is about to begin, countdown is in state "2"
      * 3 - CountingDown_1 - game is about to begin, countdown is in state "1"
      * 4 - CountingDown_0 - game is about to begin, countdown is in state "0"
      * 5 - Play - game has begun
      * 6 - End - time has ended
   * player_ids - array of playerIDs who are currently in the room
## API endpoints:
| Endpoint | Method | Request data | Response data | Possible reponse codes |
|----------|:------:|--------------|--------------|------------------------|
| /global/top/[AMOUNT_OF_TOP_PLAYERS] | GET | - | Array of player objects sorted by top_score | `200` `500` |
| /global/position/[PLAYER_ID] | GET | - | Player's position: `{ "position" : 1 }` | `200` `404` `500` |
| /global/save_score | PUT | Valid Player object | - | `200` `201` `500` |
||||||
| /room/create | GET | - | New room's ID: `{ "id" : "GR8M8" }` | `200` `500`|
| /room/connect/[ROOM_ID] | POST | Valid Player object | - | `200` `403` `500` |
| /room/disconnect/[ROOM_ID] | POST | Valid Player object | - | `200` `401` `500` |
| /room/[ROOM_ID] | GET | - | Array of a Room object indicating the current state of the room and an array of Players who are in the room | `200` `404` `500` |

## TODOS
   * Figure out a way to not have to pull the whole database and sort it in order to get the user's global position. Current implementation will definitely be very slow and painful for the server if more people join in. Also it will slow down as the time passes.
   * MAYBE security? I'm not very concerned with it as this is only meant for a game. The game has no transactions of whatever that might cause trouble when intercepted or fiddled with.
