# sshclip

- Server
  - Stores data
  - Manages client keys
  - All clients keys accepted
    - Only approved/recognized clients are allowed to put/get.
    - Clients with known keys, but not approved are immediately disconnected.
- Client
  - Tracks known server keys (hashed).
