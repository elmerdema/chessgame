import * as wasm from "hello-wasm-pack";
import { ChessGame } from "chessgame";

let chessgame= null;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const pieceSymbols = {
  1: "♙",
  2: "♘",
  3: "♗",
  4: "♖",
  5: "♕",
  6: "♔",
  7: "♟",
  8: "♞",
  9: "♝",
  10: "♜",
  11: "♛",
  12: "♚",
};
function initChess() {
  try {
    
    const chessgame = new ChessGame();
    console.log("Chess game initialized:", chessGame);
    
    drawChessboard();
}
catch (error) {
    console.error("Error initializing chess game:", error);
  }
}
function getPieceSymbol(pieceValue) {
  pieceSymbols[pieceValue] || "";
}

function drawChessboard() {
chessboardElement.innerHTML = "";
chessboardElement.style.width = `${chessboardWidth}px`;
chessboardElement.style.height = `${chessboardHeight}px`;
chessboardElement.style.display = "grid";
chessboardElement.style.gridTemplateColumns = `repeat(8, ${squareSize}px)`;
chessboardElement.style.border = "1px solid black";
chessboardElement.boxSizing = "border-box";

const boardData= chessgame.get_board();
const boardWidth = chessgame.get_board_width();
const boardHeight = chessgame.get_board_height();

if(boardWidth !== 8 || boardHeight !== 8) {
    console.error("Invalid board dimensions:", boardWidth, boardHeight);
    return;
}
for (let row =0; row <9; row++){
  for(let col=0; col <9; col++)
  {
    const index = row * boardWidth +col;
    const pieceValue = boardData[index];

    squareSize.style.backgroundColor = colors[(row + col) % 2];

    if (pieceValue !== 0) {
      const pieceElement = document.createElement("span");
      pieceElement.textContent = getPieceSymbol(pieceValue);
      pieceElement.style.fontSize = `${squareSize * 0.7}px`;
      pieceElement.style.display = "flex";
      pieceElement.style.justifyContent = "center";
      pieceElement.style.alignItems = "center";
      pieceElement.style.width = "100%";
      pieceElement.style.height = "100%"; 
      // Removed absolute positioning and transform because flexbox is used
      // TODO:  add color based on pieceValue later (e.g., if pieceValue <= 6, text color is black)
      if (pieceValue >= 1 && pieceValue <= 6) { // White pieces
          pieceElement.style.color = "black";
      } else if (pieceValue >= 7 && pieceValue <= 12) { // Black pieces
           pieceElement.style.color = "white"; // Or any color that contrasts with the square background
      }
      square.appendChild(pieceElement);
  }

  square.dataset.row = row;
  square.dataset.col = col;
  chessboardElement.appendChild(square);
}
}
}
