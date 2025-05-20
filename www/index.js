import * as wasm from "hello-wasm-pack";
import { ChessGame } from "chessgame";


let chessgame = null;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const pieceSymbols = {
  1: "♙",
  2: "♖",
  3: "♘",
  4: "♗",
  5: "♕",
  6: "♔",
  7: "♟",
  8: "♜",
  9: "♞",
  10: "♝",
  11: "♛",
  12: "♚",
};

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

      square.style.backgroundColor = colors[(row + col) % 2];

      if (pieceValue !== 0) {
        const pieceElement = document.createElement("span");
        pieceElement.textContent = getPieceSymbol(pieceValue);
        pieceElement.style.fontSize = `${squareSize * 0.7}px`;
        pieceElement.style.display = "flex";
        pieceElement.style.justifyContent = "center";
        pieceElement.style.alignItems = "center";
        pieceElement.style.width = "100%";
        pieceElement.style.height = "100%";

        if (pieceValue >= 1 && pieceValue <= 6) {
          pieceElement.style.color = "black";
        } else if (pieceValue >= 7 && pieceValue <= 12) {
          pieceElement.style.color = "white";
        }

        square.appendChild(pieceElement);
      }

      square.dataset.row = row;
      square.dataset.col = col;
      square.style.boxSizing = "border-box";
      square.addEventListener("click", onSquareClick);
      chessboardElement.appendChild(square);
    }
  }
}

function onSquareClick(event) {
  const square = event.target;
  const row = parseInt(square.dataset.row);
  const col = parseInt(square.dataset.col);
  const pieceValue = chessgame.get_piece(row, col);

  if (pieceValue !== 0) {
    const pieceElement = document.createElement("span");
    pieceElement.textContent = getPieceSymbol(pieceValue);
    pieceElement.style.fontSize = `${squareSize * 0.7}px`;
    pieceElement.style.display = "flex";
    pieceElement.style.justifyContent = "center";
    pieceElement.style.alignItems = "center";
    pieceElement.style.width = "100%";
    pieceElement.style.height = "100%";
    pieceElement.style.color = pieceValue >= 1 && pieceValue <= 6 ? "black" : "white";
    square.appendChild(pieceElement);

    const moves = chessgame.get_moves(row, col);
    console.log("Possible moves:", moves);
    moves.array.forEach(move => {
      const moveRow = move.row;
      const moveCol = move.col;
      const moveSquare = chessboardElement.querySelector(`[data-row="${moveRow}"][data-col="${moveCol}"]`);
      moveSquare.style.backgroundColor = "yellow";
      moveSquare.style.border = "2px solid red";
      moveSquare.style.boxSizing = "border-box";
      moveSquare.style.transition = "background-color 0.3s, border 0.3s";
      moveSquare.style.cursor = "pointer";
      moveSquare.addEventListener("click", () => {
        chessgame.make_move(row, col, moveRow, moveCol);
        drawChessboard();
      });
      
    });
  }
}

initChess();