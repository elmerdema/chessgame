# Chess Game

A multiplayer chess application built with Rust, WebAssembly, and Go. The game features real-time gameplay with WebSocket connections, user authentication, and a PostgreSQL database for persistent data storage.

## TODO

- improve golang project structure
- sound
- add game settings (board themes, piece sets, sound effect toggles)
- add game history





## About

This chess game combines a Rust/WebAssembly frontend for the chess logic and game board with a Go backend server handling multiplayer functionality, authentication, and data persistence. The application uses WebSockets for real-time game updates and PostgreSQL for storing user accounts and game data.

The chess engine is implemented in Rust and compiled to WebAssembly for optimal performance in the browser, while the backend API and WebSocket server are written in Go using the Gorilla Mux router.


## Running the Project

### Database Setup

Before running the backend server, you need to set up PostgreSQL.

Install PostgreSQL if you haven't already. Create a `.env` file in the root directory based on `.env.example` and set your `DATABASE_PASSWORD` to match your PostgreSQL password. Create the database by connecting to PostgreSQL using `psql` and running:

```sql
CREATE DATABASE chessgame;
```

### Starting the Application

Start the Backend Server by running `go run ./server` from the root directory. The server will start on `http://localhost:8081`.

For the Frontend Development Server, first build the wasm part using `wasm-pack build`. Navigate to the `www` directory, install dependencies with `npm install`, then start the webpack development server with `npm start`. The frontend will be available at `http://localhost:8080`.

### Accessing the Application

Open your web browser and go to `http://localhost:8080/auth.html` to log in or register. After logging in, you will be redirected to the main chess application at `http://localhost:8080/`.

## Project Structure

The project is organized into several main directories:

- `src/` - Rust source code for the chess engine and WebAssembly bindings
- `server/` - Go backend server with WebSocket handling and database operations
- `www/` - Frontend web application with JavaScript, CSS, and HTML
- `tests/` - Test files for the application

The Rust code handles chess game logic, move validation, and board state management. The Go server manages user authentication, game rooms, and real-time communication between players. The frontend provides the user interface for playing chess and interacting with other players.
