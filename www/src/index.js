import { ChessGame } from "chessgame";
import { showNotification, setOnNotificationClose, initNotificationEventListeners } from "./notification.js";
import {
    WHITE, BLACK,
    B_PAWN, B_ROOK, B_KNIGHT, B_BISHOP, B_QUEEN, B_KING,
    W_PAWN, W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING,
    pieceImageMap
} from "./pieces.js";
import { showPromotionDialog, hidePromotionDialog, setOnPromotionCompleted, setPromotionGameAndConstants } from "./promotion.js";

const API_BASE_URL = "http://localhost:8081/api";
let chessgame = null;
let currentGameID = null;
let isProcessingMove = false;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const PIECES_BASE_PATH = "./pieces/";

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

async function initChess() {
    try {
        if (!chessgame) {
            chessgame = new ChessGame();
            console.log("WASM module loaded.");
             setPromotionGameAndConstants(chessgame, {
                WHITE, BLACK,
                W_QUEEN, W_ROOK, W_KNIGHT, W_BISHOP,
                B_QUEEN, B_ROOK, B_KNIGHT, B_BISHOP
            });
        }
        
        const response = await fetch(`${API_BASE_URL}/game/new`, { method: "POST" });
        if (!response.ok) {
            throw new Error(`Failed to create new game: ${response.statusText}`);
        }
        const gameData = await response.json();

        currentGameID = gameData.gameID;
        const initialFEN = gameData.fen;
        
        console.log(`New game started with ID: ${currentGameID}`);
        console.log(`Initial FEN: ${initialFEN}`);

        chessgame.load_fen(initialFEN);

        selectedPiece = null;
        possibleMoves = [];
        isProcessingMove = false;
        
        drawChessboard();
        hidePromotionDialog();

    } catch (error) {
        console.error("Error initializing chess game:", error);
        alert("Could not connect to the server to start a new game.");
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

function coordsToAlgebraic(startRow, startCol, endRow, endCol) {
    const files = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];
    const ranks = ['8', '7', '6', '5', '4', '3', '2', '1'];

    const startFile = files[startCol];
    const startRank = ranks[startRow];
    const endFile = files[endCol];
    const endRank = ranks[endRow];

    return `${startFile}${startRank}${endFile}${endRank}`;
}

async function onSquareClick(event) {
    if (isProcessingMove || !currentGameID) {
        console.log("Ignoring click: a move is already being processed or game not started.");
        return;
    }
    if (!promotionDialog.classList.contains("hidden")) {
        console.log("Promotion dialog active, ignoring board click.");
        return;
    }

    const square = event.currentTarget;
    const row = parseInt(square.dataset.row);
    const col = parseInt(square.dataset.col);

    if (selectedPiece) {
        const startRow = selectedPiece.row;
        const startCol = selectedPiece.col;

        const isPossibleDestination = possibleMoves.some(move => move.row === row && move.col === col);

        if (isPossibleDestination) {
            isProcessingMove = true;
            const moveStr = coordsToAlgebraic(startRow, startCol, row, col);
            console.log(`Attempting to send move: ${moveStr}`);

            try {
                const response = await fetch(`${API_BASE_URL}/game/${currentGameID}/move`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ move: moveStr }),
                });

                const moveResult = await response.json();

                if (!response.ok) {
                    throw new Error(moveResult.message || 'Invalid move');
                }

                console.log("Server accepted move. New FEN:", moveResult.newFEN);
                chessgame.load_fen(moveResult.newFEN);

                const movedPiece = chessgame.get_piece(row, col);
                const pieceType = (movedPiece - 1) % 6 + 1;
                const pieceColor = getPieceColor(movedPiece);
                const isPromotion = (pieceType === 1) && ((pieceColor === WHITE && row === 0) || (pieceColor === BLACK && row === 7));

                if (isPromotion) {
                    showPromotionDialog(row, col);
                } else {
                    drawChessboard();
                    checkGameEndConditions();
                }

            } catch (error) {
                console.error("Move failed:", error);
                alert(`Invalid move: ${error.message}`);
            } finally {
                selectedPiece = null;
                possibleMoves = [];
                isProcessingMove = false;
                drawChessboard();
            }

        } else {
            selectedPiece = null;
            possibleMoves = [];
            const pieceValue = chessgame.get_piece(row, col);
            if (pieceValue !== 0 && getPieceColor(pieceValue) === chessgame.get_current_turn()) {
                handlePieceSelection(row, col);
            } else {
                drawChessboard();
            }
        }
    } else {
        const pieceValue = chessgame.get_piece(row, col);
        if (pieceValue !== 0 && getPieceColor(pieceValue) === chessgame.get_current_turn()) {
            handlePieceSelection(row, col);
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