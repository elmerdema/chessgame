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
let myPlayerColor = null; // ('white' or 'black')
let isProcessingMove = false;
let selectedPiece = null; // {row, col}
let possibleMoves = []; // Array of {row, col}

const chessboardElement = document.getElementById("chessboard");
const chessboardWidth = 400;
const chessboardHeight = 400;
const squareSize = chessboardWidth / 8;
const colors = ["#eee", "#ccc"];

const PIECES_BASE_PATH = "./pieces/";
const promotionDialog = document.getElementById("promotion-dialog");

async function initializePage() {
    const urlParams = new URLSearchParams(window.location.search);
    currentGameID = urlParams.get('gameId');
    if (!currentGameID) {
        alert('No Game ID found in URL! Returning to lobby.');
        window.location.href = '/lobby.html';
        return;
    }
    

    const socketProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socketHost = window.location.hostname + ':8081';
    const socketURL = `${socketProtocol}//${socketHost}/ws?gameId=${currentGameID}`;
    
    console.log(`Attempting to connect to unified WebSocket at: ${socketURL}`);
    
    const socket = new WebSocket(socketURL);

    socket.onopen = () => {
        console.log("Unified WebSocket connection established for game and chat:", currentGameID);
        // Enable the chat input now that the connection is open
        document.getElementById('chat-input').disabled = false;
        document.getElementById('chat-input').placeholder = "Type your message here...";
    };

    socket.onmessage = (event) => {
        const messageData = JSON.parse(event.data);
        console.log("Received data from server:", messageData);

        // Use a 'type' field to distinguish between message types
        // This requires your server to send back structured JSON
        switch (messageData.type) {
            case 'gameStateUpdate':
                // This is a game move or state sync from the server
                if (messageData.payload && messageData.payload.newFEN) {
                    chessgame.load_fen(messageData.payload.newFEN);
                    drawChessboard();
                    checkGameEndConditions();
                }
                break;
            case 'chatMessage':

                const output = document.getElementById('chat-messages');
                const p = document.createElement('p');
                p.textContent = messageData.payload;
                output.appendChild(p);
                output.scrollTop = output.scrollHeight;
                break;
            default:
                console.warn("Received unknown message type:", messageData.type);
        }
    };

    socket.onclose = (event) => {
        console.log("WebSocket connection closed.", event.reason);
        // Disable the chat input if the connection is lost
        const chatInput = document.getElementById('chat-input');
        chatInput.disabled = true;
        chatInput.placeholder = "Connection closed.";
    };

    socket.onerror = (error) => {
        console.error("WebSocket error:", error);
    };

    window.sendMessage = function(event) {
        const messageInput = document.getElementById('chat-input');
        if (event.key === 'Enter' && !messageInput.disabled && messageInput.value.trim() !== '') {

            const chatPayload = {
                type: 'chat',
                payload: messageInput.value
            };
            socket.send(JSON.stringify(chatPayload));
            messageInput.value = ''; 
        }
    };

    
    setupWasmAndListeners();
    
    try {
        const gameResponse = await fetch(`${API_BASE_URL}/game/${currentGameID}`, { credentials: 'include' });
        if (!gameResponse.ok) {
            throw new Error(`Could not fetch game data: ${await gameResponse.text()}`);
        }
        const gameData = await gameResponse.json();
        
        myPlayerColor = gameData.playerColor; 
        chessgame.load_fen(gameData.fen);
        
        console.log(`Loaded game ${currentGameID}. Your color: ${myPlayerColor}.`);
        document.title = `Chess Game - ${currentGameID}`;
        
        drawChessboard();
        hidePromotionDialog();
    } catch (error) {
        console.error(error);
        alert(error.message);
        window.location.href = '/lobby.html';
    }
}

function setupWasmAndListeners() {
    if (chessgame) return;

    chessgame = new ChessGame();
    console.log("WASM module loaded.");
    
    setPromotionGameAndConstants(chessgame, {
        WHITE, BLACK,
        W_QUEEN, W_ROOK, W_KNIGHT, W_BISHOP,
        B_QUEEN, B_ROOK, B_KNIGHT, B_BISHOP
    });

    // When a promotion is chosen in the dialog, sync it with the server
    setOnPromotionCompleted((startRow, startCol, endRow, endCol, promotionChar) => {
        syncMoveWithServer(startRow, startCol, endRow, endCol, promotionChar);
    });

    // When the game-over notification is closed, redirect to the lobby
    setOnNotificationClose(() => { window.location.href = '/lobby.html'; });
    initNotificationEventListeners();
}

export function getPieceImagePath(pieceValue) {
    const imageName = pieceImageMap[pieceValue];
    return imageName ? PIECES_BASE_PATH + imageName : "";
}

function getPieceColor(pieceValue) {
    if (pieceValue >= W_PAWN && pieceValue <= W_KING) return WHITE;
    if (pieceValue >= B_PAWN && pieceValue <= B_KING) return BLACK;
    return 0;
}

function drawChessboard() {
    if (!chessgame) return;
    chessboardElement.innerHTML = "";
    chessboardElement.style.width = `${chessboardWidth}px`;
    chessboardElement.style.height = `${chessboardHeight}px`;
    chessboardElement.style.display = "grid";
    chessboardElement.style.gridTemplateColumns = `repeat(8, ${squareSize}px)`;
    chessboardElement.style.gridTemplateRows = `repeat(8, ${squareSize}px)`;
    const boardData = chessgame.get_board();

    const isBlackView = myPlayerColor === 'black';

    for (let visualRow = 0; visualRow < 8; visualRow++) {
        for (let visualCol = 0; visualCol < 8; visualCol++) {
            const row = isBlackView ? 7 - visualRow : visualRow;
            const col = isBlackView ? 7 - visualCol : visualCol;

            const square = document.createElement("div");
            const pieceValue = boardData[row * 8 + col];
            let bgColor = colors[(row + col) % 2];
            let borderColor = "none";
            let cursorStyle = "default";

            if (selectedPiece && selectedPiece.row === row && selectedPiece.col === col) {
                borderColor = "3px solid blue";
            }
            if (possibleMoves.some(move => move.row === row && move.col === col)) {
                bgColor = "rgba(0, 255, 0, 0.5)";
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
    return `${files[startCol]}${ranks[startRow]}${files[endCol]}${ranks[endRow]}`;
}

async function onSquareClick(event) {
    if (isProcessingMove || !currentGameID || !promotionDialog.classList.contains("hidden")) {
        return;
    }
    
    // Client-side turn enforcement for better UX
    const currentTurnColor = chessgame.get_current_turn() === WHITE ? 'white' : 'black';
    if (myPlayerColor !== currentTurnColor) {
        return;
    }

    const square = event.currentTarget;
    const row = parseInt(square.dataset.row);
    const col = parseInt(square.dataset.col);

    if (selectedPiece) {
        const startRow = selectedPiece.row;
        const startCol = selectedPiece.col;

        if (possibleMoves.some(move => move.row === row && move.col === col)) {
            isProcessingMove = true;
            const movingPiece = { startRow, startCol, endRow: row, endCol: col };
            selectedPiece = null;
            possibleMoves = [];
            
            try {
                const isPromotion = await chessgame.make_move(movingPiece.startRow, movingPiece.startCol, movingPiece.endRow, movingPiece.endCol);
                drawChessboard();
                if (isPromotion) {
                    showPromotionDialog(movingPiece.startRow, movingPiece.startCol, movingPiece.endRow, movingPiece.endCol);
                } else {
                    await syncMoveWithServer(movingPiece.startRow, movingPiece.startCol, movingPiece.endRow, movingPiece.endCol);
                }
            } catch (error) {
                console.error("WASM move validation failed:", error);
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
        handlePieceSelection(row, col);
    }
}

async function syncMoveWithServer(startRow, startCol, endRow, endCol, promotionChar = '') {
    const moveStr = coordsToAlgebraic(startRow, startCol, endRow, endCol) + promotionChar;
    
    try {
        const response = await fetch(`${API_BASE_URL}/game/${currentGameID}/move`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ move: moveStr }),
            credentials: 'include'
        });

        const moveResult = await response.json();
        if (!response.ok) {
            throw new Error(moveResult.message || 'Server rejected a locally-validated move!');
        }
        
        // The server is the source of truth. It will broadcast the new FEN via WebSocket.
        // We can optionally re-sync here, but the WebSocket message is the primary mechanism.
        chessgame.load_fen(moveResult.newFEN);
        checkGameEndConditions();

    } catch (error) {
        console.error("Failed to sync move with server:", error);
        alert(`A critical error occurred: ${error.message}. Please refresh.`);
    } finally {
        isProcessingMove = false;
        drawChessboard();
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
    }
    drawChessboard();
}

function warnKingCheck() {
    if (chessgame.check()) {
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

// starting point for the appplication
initializePage();