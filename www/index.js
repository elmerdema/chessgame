import * as wasm from "hello-wasm-pack";
import { ChessGame } from "chessgame";

function initChess() {
  try {
    wasm.greet();
    
    const chessGame = new ChessGame();
    
    // Check if the game is properly initialized
    console.log("Chess game initialized:", chessGame);
    
    
    //JSON board
    try {
      const boardJson = chessGame.get_board_json();
      
      const board = JSON.parse(boardJson);
      console.log("Parsed board:", board);
    } catch (jsonError) {
      console.error("Failed to get board JSON:", jsonError);
    }
  } catch (gameError) {
    console.error("Failed to initialize chess game:", gameError);
  }
}

initChess();