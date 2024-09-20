# gRPC streaming ping pong


## 1.0.0 
- working gRPC streaming chat client server application
- slog logger

## 1.0.1
- added client / server tls communication
- added logger / auth client / server interceptors

## 1.0.2
- added mutual tls, client now has its own certificate and server has to validate it

### future plains
- [ ] add server calling rest service, 
use json decoder in streaming mode to read json data and converts it to protobuf response 
