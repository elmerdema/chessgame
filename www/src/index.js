import { ChessGame } from "chessgame";
import { showNotification, setOnNotificationClose, initNotificationEventListeners } from "./notification.js";

let chessgame = null;

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const WHITE = 1;
const BLACK = 2;

const B_PAWN = 7;
const B_ROOK = 8;
const B_KNIGHT = 9;
const B_BISHOP = 10;
const B_QUEEN = 11;
const B_KING = 12;

const W_PAWN = 1;
const W_ROOK = 2;
const W_KNIGHT = 3;
const W_BISHOP = 4;
const W_QUEEN = 5;
const W_KING = 6;

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

let promotionPawnCoords = null; 

// DOM elements for promotion dialog
const promotionOverlay = document.getElementById("promotion-overlay");
const promotionDialog = document.getElementById("promotion-dialog");
const promotionChoicesContainer = document.getElementById("promotion-choices");

function initChess() {
    try {
        chessgame = new ChessGame();
        console.log("Chess game initialized:", chessgame);
        drawChessboard();
        hidePromotionDialog();
    } catch (error) {
        console.error("Error initializing chess game:", error);
    }
}

function getPieceImagePath(pieceValue) {
    const imageName = pieceImageMap[pieceValue];
    return imageName ? PIECES_BASE_PATH + imageName : "";
}

function getPieceColor(pieceValue) {
    // White pieces values (1-6)
    if (pieceValue >= W_PAWN && pieceValue <= W_KING) {
        return WHITE;
    }
    // Black pieces values (7-12)
    if (pieceValue >= B_PAWN && pieceValue <= B_KING) {
        return BLACK;
    }
    return 0; // Empty square or invalid piece
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
                const pieceName = pieceImageMap[pieceValue].replace(/(black-|white-|\.png)/g, ''); // Extracts 'pawn', 'rook', etc.
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
                drawChessboard();

                if (promotionInfo && promotionInfo.length === 2) {
                    promotionPawnCoords = { x: promotionInfo[0], y: promotionInfo[1] };
                    showPromotionDialog();
                } else {
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
    // Check if movesArray is array-like (Array, Uint8Array, etc.) and has an even number of elements, 
    //earlier this throwed always false for some reason?
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

function showPromotionDialog() {
    promotionChoicesContainer.innerHTML = "";

    const playerColor = chessgame.get_current_turn();

    const promotionPieces = [
        { type: "Queen", value: playerColor === WHITE ? W_QUEEN : B_QUEEN },
        { type: "Rook", value: playerColor === WHITE ? W_ROOK : B_ROOK },
        { type: "Knight", value: playerColor === WHITE ? W_KNIGHT : B_KNIGHT },
        { type: "Bishop", value: playerColor === WHITE ? W_BISHOP : B_BISHOP },
    ];

    promotionPieces.forEach(piece => {
        const choiceDiv = document.createElement("div");
        choiceDiv.classList.add("promotion-choice");

        const pieceImage = document.createElement("img");
        pieceImage.src = getPieceImagePath(piece.value);
        pieceImage.alt = piece.type;
        pieceImage.style.pointerEvents = 'none';

        choiceDiv.appendChild(pieceImage);
        choiceDiv.dataset.pieceValue = piece.value;
        choiceDiv.dataset.pieceType = piece.type;
        choiceDiv.addEventListener("click", onPromotionChoice);
        promotionChoicesContainer.appendChild(choiceDiv);
    });

    promotionOverlay.classList.remove("hidden");
    promotionOverlay.classList.add("visible");
    promotionDialog.classList.remove("hidden");
    promotionDialog.classList.add("visible");
    promotionDialog.focus();
}

function hidePromotionDialog() {
    promotionOverlay.classList.remove("visible");
    promotionOverlay.classList.add("hidden");
    promotionDialog.classList.remove("visible");
    promotionDialog.classList.add("hidden");
    promotionPawnCoords = null;
}

async function onPromotionChoice(event) {
    const chosenPieceValue = parseInt(event.currentTarget.dataset.pieceValue);
    const chosenPieceType = event.currentTarget.dataset.pieceType;

    if (!promotionPawnCoords) {
        console.error("No pawn promotion pending, but choice was made!");
        hidePromotionDialog();
        return;
    }

    const { x, y } = promotionPawnCoords;

    try {
        await chessgame.promote_pawn(x, y, chosenPieceValue);
        console.log(`Pawn at (${x},${y}) promoted to ${chosenPieceType}.`);

        hidePromotionDialog();
        drawChessboard();

        chessgame.change_turn();
        checkGameEndConditions();

    } catch (error) {
        console.error("Error promoting pawn:", error);
        alert(`Promotion failed: ${error.message || error}`);
        hidePromotionDialog();
        drawChessboard();
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

setOnNotificationClose(initChess);
initNotificationEventListeners();

initChess();