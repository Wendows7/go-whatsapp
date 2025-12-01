## Go Whatsapp
Go Whatsapp is my learning golang project, build with go language and use whatsmeow library that provides an interface to interact with WhatsApp Web. It allows developers to create applications that can send and receive messages, manage contacts, and perform various other actions on WhatsApp.

## Features
- Send text messages to individual contacts or groups
- Send alerting messages based by prometheus alertmanager structure data

## Dependencies
- Go 1.25 or higher
- PostgreSQL database

## Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/Wendows7/go-whatsapp.git
   cd go-whatsapp
2. Install the required dependencies:
   ```bash
    go mod tidy
    ```
3. Build the project:
    ```bash
     cp app.example.env app.env
     go build -o go-whatsapp main.go
     ```
4. Run the application:
    ```bash
     ./go-whatsapp
     ```

## Running with Docker
1. Build the Docker image:
   ```bash
   docker build -t go-whatsapp .
   ```
2. Run the Docker container:
   ```bash
   docker compose up -d
    ```
   
## Configuration
- Rename `app.example.env` to `app.env` and modify the configuration variables as needed.
- Set up the PostgreSQL database and update the connection details in the `app.env` file.

## Usage
- /api/send-message : to send text message to whatsapp number

  example payload to whatsapp number:
  ```json
  {
    "number": "6281234567890",
    "message": "Hello, this is a test message from Go Whatsapp!",
    "token_key": "your_secret"
  }
  ```
  example payload to whatsapp group:
  ```json
  {
    "number": "123123123@g.us",
    "message": "Hello, this is a test message to the group from Go Whatsapp!",
    "token_key": "your_secret"
  }
    ```
- /api/send-alert?token_key=123&?number=6281222123123 : to send alert message based on prometheus alertmanager structure data, example payload:
  ```json
  {
    "receiver": "whatsapp-notifications",
    "status": "firing",
    "alerts": [
      {
        "status": "firing",
        "labels": {
          "alertname": "HighCPUUsage",
          "instance": "server1.example.com",
          "severity": "critical"
        },
        "annotations": {
          "summary": "CPU usage is above 90% on server1.example.com",
          "description": "The CPU usage has been above 90% for the last 5 minutes."
        },
        "startsAt": "2023-10-01T12:00:00Z",
        "endsAt": "0001-01-01T00:00:00Z",
        "generatorURL": "http://prometheus.example.com/graph?g0.expr=cpu_usage%3E90&g0.tab=1"
      }
    ]
  }
  ```