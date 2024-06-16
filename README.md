# gRPC streaming ping pong


## 1.0.0 
- working gRPC streaming chat client server application
- slog logger

## 1.0.1
- added client / server tls communication
- added logger / auth client / server interceptors

### future plains
- Implement mutual TLS. At the moment, the server has already shared its certificate with the client. For mutual TLS, the client also has to share its certificate with the server. So we will update cert/gen.sh script to create and sign a certificate for the client.