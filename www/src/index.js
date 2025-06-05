import { ChessGame } from "chessgame";
import { showNotification, setOnNotificationClose, initNotificationEventListeners } from "./notification.js";

import { showPromotionDialog, hidePromotionDialog, setOnPromotionCompleted, setPromotionGameAndConstants } from "./promotion.js";

let chessgame = null;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

export const WHITE = 1;
export const BLACK = 2;

export const B_PAWN = 7;
export const B_ROOK = 8;
export const B_KNIGHT = 9;
export const B_BISHOP = 10;
export const B_QUEEN = 11;
export const B_KING = 12;

export const W_PAWN = 1;
export const W_ROOK = 2;
export const W_KNIGHT = 3;
export const W_BISHOP = 4;
export const W_QUEEN = 5;
export const W_KING = 6;

const PIECES_BASE_PATH = "./pieces/";

const pieceImageMap = {
    [B_PAWN]: "black-pawn.png",
    [B_ROOK]: "black-rook.png",
    [B_KNIGHT]: "black-knight.png",
    [B_BISHOP]: "black-bishop.png",
    [B_QUEEN]: "black-queen.png",
    [B_KING]: "black-king.png",
    [W_PAWN]: "white-pawn.png",
    [W_ROOK]: "white-rook.png",
    [W_KNIGHT]: "white-knight.png",
    [W_BISHOP]: "white-bishop.png",
    [W_QUEEN]: "white-queen.png",
    [W_KING]: "white-king.png",
};

let selectedPiece = null;
let possibleMoves = [];

const promotionDialog = document.getElementById("promotion-dialog");

export function getPieceImagePath(pieceValue) {
    const imageName = pieceImageMap[pieceValue];
    return imageName ? PIECES_BASE_PATH + imageName : "";
}

function getPieceColor(pieceValue) {
    if (pieceValue >= W_PAWN && pieceValue <= W_KING) {
        return WHITE;
    }
    if (pieceValue >= B_PAWN && pieceValue <= B_KING) {
        return BLACK;
    }
    return 0;
}

function initChess() {
    try {
        chessgame = new ChessGame();
        console.log("Chess game initialized:", chessgame);
        setPromotionGameAndConstants(chessgame, {
            WHITE, BLACK,
            W_QUEEN, W_ROOK, W_KNIGHT, W_BISHOP,
            B_QUEEN, B_ROOK, B_KNIGHT, B_BISHOP
        });
        drawChessboard();
        hidePromotionDialog();
    } catch (error) {
        console.error("Error initializing chess game:", error);
    }
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

            const isPossibleDestination = possibleMoves.some(move => move.row === row && move.col === col);
            if (isPossibleDestination) {
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
            square.style.display = "flex";
            square.style.justifyContent = "center";
            square.style.alignItems = "center";

            if (pieceValue !== 0) {
                const pieceElement = document.createElement("img");
                pieceElement.src = getPieceImagePath(pieceValue);
                const pieceName = pieceImageMap[pieceValue].replace(/(black-|white-|\.png)/g, '');
                pieceElement.alt = `${getPieceColor(pieceValue) === WHITE ? "White" : "Black"} ${pieceName}`;
                pieceElement.style.pointerEvents = 'none';

                square.appendChild(pieceElement);
            }

            square.dataset.row = row;
            square.dataset.col = col;
            square.addEventListener("click", onSquareClick);

            chessboardElement.appendChild(square);
        }
    }
    warnKingCheck();
}

async function onSquareClick(event) {
    if (!promotionDialog.classList.contains("hidden")) {
        console.log("Promotion dialog active, ignoring board click.");
        return;
    }

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
                const promotionInfo = await chessgame.make_move(startRow, startCol, row, col);
                console.log(`Move successful: ${startRow},${startCol} to ${row},${col}. Promotion info:`, promotionInfo);

                selectedPiece = null;
                possibleMoves = [];

                if (promotionInfo && promotionInfo.length === 2) {
                    // Pass the promotion pawn's coordinates to showPromotionDialog
                    showPromotionDialog(promotionInfo[0], promotionInfo[1]);
                } else {
                    drawChessboard();
                    checkGameEndConditions();
                }

            } catch (error) {
                console.error("Move failed:", error);
                selectedPiece = null;
                possibleMoves = [];
                drawChessboard();
                alert(`Invalid move: ${error.message || error}`);
            }

        } else {
            console.log("Clicked outside valid moves for the selected piece. Deselecting current and checking new square.");
            selectedPiece = null;
            possibleMoves = [];

            const pieceValueAtClickedSquare = chessgame.get_piece(row, col);
            const currentTurn = chessgame.get_current_turn();
            const pieceColorAtClickedSquare = getPieceColor(pieceValueAtClickedSquare);

            console.log(`Checking new square for selection: (${row},${col}), Value: ${pieceValueAtClickedSquare}, Color: ${pieceColorAtClickedSquare}, Current Turn: ${currentTurn}`);

            if (pieceValueAtClickedSquare !== 0 && pieceColorAtClickedSquare === currentTurn) {
                handlePieceSelection(row, col);
            } else {
                console.log("New square is empty or not your piece. Board redrawn, nothing selected.");
                drawChessboard();
            }
        }

    } else {
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
        const kingPosition = chessgame.get_king_position();
        if (kingPosition && kingPosition.length === 2) {
            const kingSquare = document.querySelector(`[data-row="${kingPosition[0]}"][data-col="${kingPosition[1]}"]`);
            if (kingSquare) {
                kingSquare.style.backgroundColor = "rgba(255, 0, 0, 0.5)";
            }
        }
    }
}


function checkGameEndConditions() {
    if (chessgame.is_stalemate()) {
        showNotification("draw");
    } else if (chessgame.checkmate()) {
        const winnerPlayer = chessgame.get_current_turn() === BLACK ? "White" : "Black";
        showNotification(winnerPlayer);
    }
}

setOnPromotionCompleted(() => {
    drawChessboard();
    checkGameEndConditions();
});

setOnNotificationClose(initChess);
initNotificationEventListeners();

initChess();