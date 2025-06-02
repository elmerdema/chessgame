import * as wasm from "hello-wasm-pack";
import { ChessGame } from "chessgame";
import {showCheckmateNotification, setOnNotificationClose, initNotificationEventListeners} from "./notification.js";
let chessgame = null;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const WHITE = 2;
const BLACK = 1;

const pieceSymbols = {
  1: "♟", 2: "♜", 3: "♞", 4: "♝", 5: "♛", 6: "♚", // Black pieces
  7: "♟", 8: "♖", 9: "♘", 10: "♗", 11: "♕", 12: "♔", // White pieces
};

let selectedPiece = null;
let possibleMoves = [];

function initChess() {
  try {
    chessgame = new ChessGame();
    console.log("Chess game initialized:", chessgame);
    drawChessboard();
  } catch (error) {
    console.error("Error initializing chess game:", error);
  }
}

function getPieceSymbol(pieceValue) {
  return pieceSymbols[pieceValue] || "";
}

function getPieceColor(pieceValue) {
    if (pieceValue >= 1 && pieceValue <= 6) {
        return BLACK;
    }
    if (pieceValue >= 7 && pieceValue <= 12) {
        return WHITE;
    }
    return 0;
}

function drawChessboard() {
  if (!chessgame) {
    console.error("Chess game not initialized. Cannot draw board.");
    return;
  }

  chessboardElement.innerHTML = "";
  chessboardElement.style.width = `${chessboardWidth}px`;
  chessboardElement.style.height = `${chessboardHeight}px`;
  chessboardElement.style.display = "grid";
  chessboardElement.style.gridTemplateColumns = `repeat(8, ${squareSize}px)`;
  chessboardElement.style.gridTemplateRows = `repeat(8, ${squareSize}px)`;
  chessboardElement.style.border = "1px solid black";
  chessboardElement.style.boxSizing = "border-box";

  const boardData = chessgame.get_board();
  const boardWidth = chessgame.get_board_width();
  const boardHeight = chessgame.get_board_height();

  if (boardWidth !== 8 || boardHeight !== 8) {
    console.error("Invalid board dimensions:", boardWidth, boardHeight);
    return;
  }

  for (let row = 0; row < 8; row++) {
    for (let col = 0; col < 8; col++) {
      const square = document.createElement("div");

      const index = row * boardWidth + col;
      const pieceValue = boardData[index];

      let bgColor = colors[(row + col) % 2];
      let borderColor = "none";
      let cursorStyle = "default";


      if (selectedPiece && selectedPiece.row === row && selectedPiece.col === col) {
          borderColor = "3px solid blue";
      }

      const isPossibleMove = possibleMoves.some(move => move.row === row && move.col === col);
      if (isPossibleMove) {
          // If it's a capture, maybe differnet color (e.g., red-ish green)?? 
          const pieceAtDestination = chessgame.get_piece(row, col);
          if (pieceAtDestination !== 0 && getPieceColor(pieceAtDestination) !== chessgame.get_current_turn()) {
              bgColor = "rgba(255, 100, 100, 0.6)";
              borderColor = "2px solid red";
          } else {
              bgColor = "rgba(0, 255, 0, 0.5)";
              borderColor = "2px solid green";
          }
          cursorStyle = "pointer";
      }


      square.style.backgroundColor = bgColor;
      square.style.border = borderColor;
      square.style.boxSizing = "border-box";
      square.style.cursor = cursorStyle;

      if (pieceValue !== 0) {
        const pieceElement = document.createElement("span");
        pieceElement.textContent = getPieceSymbol(pieceValue);
        pieceElement.style.fontSize = `${squareSize * 0.7}px`;
        pieceElement.style.display = "flex";
        pieceElement.style.justifyContent = "center";
        pieceElement.style.alignItems = "center";
        pieceElement.style.width = "100%";
        pieceElement.style.height = "100%";
        pieceElement.style.color = getPieceColor(pieceValue) === WHITE ? "black" : "white";
        pieceElement.style.pointerEvents = 'none'; // Prevents piece clicks from interfering with square clicks

        square.appendChild(pieceElement);
      }

      square.dataset.row = row;
      square.dataset.col = col;
      square.addEventListener("click", onSquareClick);

      chessboardElement.appendChild(square);
    }
  }
  warnKingCheck(); // Check if the king is in check after selecting a piece
}


function onSquareClick(event) {
  const square = event.currentTarget;
  const row = parseInt(square.dataset.row);
  const col = parseInt(square.dataset.col);

  console.log(`Clicked square: ${row}, ${col}`);

  if (selectedPiece) {
      const startRow = selectedPiece.row;
      const startCol = selectedPiece.col;

      const isPossibleDestination = possibleMoves.some(move => move.row === row && move.col === col);

      if (isPossibleDestination) {
          try {
              chessgame.make_move(startRow, startCol, row, col);
              console.log(`Move successful: ${startRow},${startCol} to ${row},${col}`);
              selectedPiece = null;
              possibleMoves = [];
              drawChessboard();
              checkmateDetection(); // checkmate detection after every move

          } catch (error) {
              console.error("Move failed:", error);
              selectedPiece = null;
              possibleMoves = [];
              drawChessboard(); // Redraw to clear highlights if move fails
          }

      } else {
          // Clicked on a square that is not a valid move for the selected piece.
          // This could be another piece of the same color, an opponent's piece (not a capture), or an empty square.
          console.log("Clicked outside valid moves for the selected piece. Deselecting current and checking new square.");
          selectedPiece = null; // Deselect current piece
          possibleMoves = [];   // Clear its moves

          const pieceValueAtClickedSquare = chessgame.get_piece(row, col);
          const currentTurn = chessgame.get_current_turn();
          const pieceColorAtClickedSquare = getPieceColor(pieceValueAtClickedSquare);

          console.log(`Checking new square for selection: (${row},${col}), Value: ${pieceValueAtClickedSquare}, Color: ${pieceColorAtClickedSquare}, Current Turn: ${currentTurn}`);

          if (pieceValueAtClickedSquare !== 0 && pieceColorAtClickedSquare === currentTurn) {
              handlePieceSelection(row, col); // Select the new piece
          } else {
               console.log("New square is empty or not your piece. Board redrawn, nothing selected.");
               drawChessboard();
          }
      }

  } else { // No piece is currently selected
      const pieceValueAtClickedSquare = chessgame.get_piece(row, col);
      const currentTurn = chessgame.get_current_turn();
      const pieceColorAtClickedSquare = getPieceColor(pieceValueAtClickedSquare);

      if (pieceValueAtClickedSquare !== 0 && pieceColorAtClickedSquare === currentTurn) {
          handlePieceSelection(row, col);
      } else {
            console.log("Cannot select: Empty square or not your piece.");
        }
    }
}

function handlePieceSelection(row, col) {
  selectedPiece = { row, col };

  const movesArray = chessgame.get_moves(row, col);

  possibleMoves = [];
  // Check if movesArray is array-like (Array, Uint8Array, etc.) and has an even number of elements, 
  //arlier this throwed always false for some reason?
  if (movesArray && typeof movesArray.length === 'number' && movesArray.length % 2 === 0) {
      for (let i = 0; i < movesArray.length; i += 2) {
          possibleMoves.push({ row: movesArray[i], col: movesArray[i + 1] });
      }
  } else {
      if (movesArray === null || movesArray === undefined) {
          console.warn(`get_moves for (${row},${col}) returned null or undefined.`);
      } else if (typeof movesArray.length !== 'number' || movesArray.length % 2 !== 0) {
          console.warn(`get_moves for (${row},${col}) returned an array-like object with invalid length: ${movesArray.length}. Content:`, movesArray);
      } else {
          console.warn(`get_moves for (${row},${col}) did not return a valid flat array or array-like structure. Received:`, movesArray);
      }
  }

  console.log(`Selected piece at ${row},${col}. Found ${possibleMoves.length} possible moves.`);

  drawChessboard();
}

function warnKingCheck() {
  const isInCheck = chessgame.check();
  const currentTurn = chessgame.get_current_turn();

  if (isInCheck) {

    const kingPosition = chessgame.get_king_position(currentTurn); 
    if (kingPosition) {
        const kingSquare = document.querySelector(`[data-row="${kingPosition[0]}"][data-col="${kingPosition[1]}"]`);
        if (kingSquare) {
            kingSquare.style.backgroundColor = "rgba(255, 0, 0, 0.5)"; // Highlight the king in check
        }
    }
  }
}

function checkmateDetection() {
  const isCheckmate = chessgame.checkmate();
  if (isCheckmate) {
    const winnerPlayer = chessgame.get_current_turn() === WHITE ? "Black" : "White"; 
    
    showCheckmateNotification(winnerPlayer);
  }
}
setOnNotificationClose(initChess); // we reset the chess game when the notification is closed
initNotificationEventListeners();


initChess();