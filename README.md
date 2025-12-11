# Chatterbox

Chatterbox is a fully open source, free, and end-to-end (E2E) encrypted chat application. It is designed to be lightning fast and run anywhere, providing a secure communication platform for everyone.

## Features

-   **Open Source:** Free to use, modify, and distribute.
-   **E2E Encrypted:** Your messages are secure and can only be read by you and the recipient.
-   **Lightning Fast:** Optimized for speed and performance.
-   **Cross-Platform:** Runs on any device with a modern web browser.

## Architecture

The project is divided into two main components:

-   **Server:** A Go-based backend that handles synchronization and message storage using Google Cloud Spanner.
-   **Client:** A React (Vite) frontend that provides the user interface and handles local storage (IndexedDB).

## Prerequisites

Before you begin, ensure you have the following installed:

-   **Go:** (v1.24.0 or later) [Download Go](https://go.dev/dl/)
-   **Node.js & npm:** [Download Node.js](https://nodejs.org/)
-   **Google Cloud Platform (GCP) Project:**
    -   Enable the **Spanner API**.
    -   Create a Spanner Instance named `chatterbox-db`.
    -   Create a Database within that instance.

## Installation & Setup

### 1. Server Setup

The server is responsible for syncing messages and rooms. It connects to Google Cloud Spanner.

1.  Navigate to the server directory:
    ```bash
    cd server
    ```

2.  Install Go dependencies:
    ```bash
    go mod download
    ```

3.  Set the required environment variables:
    ```bash
    export GOOGLE_CLOUD_PROJECT=your-project-id
    export SPANNER_DATABASE=your-database-id
    ```
    *(Optional) Set a custom port (default is 8080):*
    ```bash
    export PORT=8080
    ```

4.  Run the server:
    ```bash
    go run main.go
    ```
    You should see a message indicating the server is listening on the specified port.

### 2. Client Setup

The client is the web interface for Chatterbox.

1.  Navigate to the client directory:
    ```bash
    cd client
    ```

2.  Install npm dependencies:
    ```bash
    npm install
    ```

3.  Configure the environment:
    -   Copy the example environment file:
        ```bash
        cp .env.example .env
        ```
    -   Edit `.env` and set `VITE_API_URL` to your server's URL (e.g., `http://localhost:8080/api` if running locally).

4.  Start the development server:
    ```bash
    npm run dev
    ```

5.  Open your browser and navigate to the URL shown in the terminal (usually `http://localhost:5173`).

## Development

-   **Server:** Located in `server/`. The main entry point is `main.go`. Routes are defined in `server/main.go`.
-   **Client:** Located in `client/`. It uses Vite for building and development. The main logic for syncing and messages is in `client/src/sync.ts`.

## License

[Add License Information Here]
