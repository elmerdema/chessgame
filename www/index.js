import * as wasm from "hello-wasm-pack";
import { ChessGame } from "chessgame";

// No need to await a default export since it doesn't exist
function initChess() {
  try {
    // Greet from hello-wasm-pack (this function exists according to the error message)
    wasm.greet();
    
    // Initialize the chess game
    const chessGame = ChessGame.new();
    
    // Check if the game is properly initialized
    console.log("Chess game initialized:", chessGame);
    
    // Try to get the board using the flat array method first
    try {
      const boardArray = chessGame.get_board();
      console.log("Chess board (flat array):", boardArray);
      
      // Format the flat array as 2D for display
      const width = chessGame.get_board_width();
      const height = chessGame.get_board_height();
      const board2D = [];
      
      for (let i = 0; i < height; i++) {
        const row = [];
        for (let j = 0; j < width; j++) {
          row.push(boardArray[i * width + j]);
        }
        board2D.push(row);
      }
      
      console.log("Chess board (2D):", board2D);
    } catch (arrayError) {
      console.error("Failed to get board array:", arrayError);
    }
    
    // Try to get the JSON board
    try {
      const boardJson = chessGame.get_board_json();
      console.log("Raw board JSON:", boardJson);
      
      const board = JSON.parse(boardJson);
      console.log("Parsed board:", board);
    } catch (jsonError) {
      console.error("Failed to get board JSON:", jsonError);
    }
  } catch (gameError) {
    console.error("Failed to initialize chess game:", gameError);
  }
}

// Call the initialization function
initChess();