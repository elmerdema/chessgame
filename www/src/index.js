import { ChessGame } from "chessgame";
import { showNotification, setOnNotificationClose, initNotificationEventListeners } from "./notification.js";
import {
    WHITE, BLACK,
    B_PAWN, B_ROOK, B_KNIGHT, B_BISHOP, B_QUEEN, B_KING,
    W_PAWN, W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING,
    pieceImageMap
} from "./pieces.js";
import { showPromotionDialog, hidePromotionDialog, setOnPromotionCompleted, setPromotionGameAndConstants } from "./promotion.js";
import { loadAllSounds, play } from "./sound.js";
const API_BASE_URL = "/api";
let chessgame = null;
let currentGameID = null;
let myPlayerColor = null; // ('white' or 'black')
let isProcessingMove = false;
let selectedPiece = null; // {row, col}
let possibleMoves = []; // Array of {row, col}
let moveHistory = []; // Array of {move: string, notation: string, player: string, timestamp: Date}
let moveNumber = 1;

const chessboardElement = document.getElementById("chessboard");

const PIECES_BASE_PATH = "./pieces/";
const promotionDialog = document.getElementById("promotion-dialog");

async function initializePage() {
    await loadAllSounds();
    const urlParams = new URLSearchParams(window.location.search);
    currentGameID = urlParams.get('gameId');
    if (!currentGameID) {
        alert('No Game ID found in URL! Returning to lobby.');
        window.location.href = '/lobby.html';
        return;
    }
    

    const socketProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const socketHost = window.location.hostname + ':8081';
    const socketURL = `${socketProtocol}//${socketHost}/api/ws?gameId=${currentGameID}`;
    
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

        switch (messageData.type) {
            case 'gameStateUpdate':
                if (messageData.payload && messageData.payload.newFEN) {
                    // Only update if the FEN is different to avoid redundant redraws
                    if (chessgame.fen() !== messageData.payload.newFEN) {
                        console.log("Applying game state update from server.");
                        chessgame.load_fen(messageData.payload.newFEN);
                        
                        if (messageData.payload.move) {
                            addMoveToHistory(messageData.payload.move, messageData.payload.player);
                        }
                        play("move")
                        drawChessboard();
                        updateGameStatus();
                        checkGameEndConditions();
                    }
                }
                break;
            case 'chat':
                play("notification");
                const output = document.getElementById('chat-messages');
                const messageDiv = document.createElement('div');
                messageDiv.className = 'chat-message';
                
                const messageContent = document.createElement('div');
                messageContent.className = 'message-content';
                messageContent.textContent = messageData.payload;
                
                const timestamp = document.createElement('div');
                timestamp.className = 'message-timestamp';
                const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
                timestamp.textContent = time;
                
                const sender = document.createElement('div');
                sender.className = 'message-sender';
                sender.textContent = messageData.sender || 'Opponent';
                
                messageDiv.appendChild(sender);
                messageDiv.appendChild(messageContent);
                messageDiv.appendChild(timestamp);
                
                output.appendChild(messageDiv);
                output.scrollTop = output.scrollHeight;

                break;
            default:
                console.warn("Received unknown message type:", messageData.type, messageData);
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
                payload: messageInput.value,
                gameID: currentGameID
            };
            socket.send(JSON.stringify(chatPayload));
            
            const output = document.getElementById('chat-messages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'chat-message own';
            
            const messageContent = document.createElement('div');
            messageContent.className = 'message-content';
            messageContent.textContent = messageInput.value;
            
            const timestamp = document.createElement('div');
            timestamp.className = 'message-timestamp';
            const time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            timestamp.textContent = time;
            
            const sender = document.createElement('div');
            sender.className = 'message-sender';
            sender.textContent = 'You';
            
            messageDiv.appendChild(sender);
            messageDiv.appendChild(messageContent);
            messageDiv.appendChild(timestamp);
            
            output.appendChild(messageDiv);
            output.scrollTop = output.scrollHeight;
            
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
        play("start")
        
        if (gameData.moveHistory) {
            moveHistory = gameData.moveHistory;
            updateGameHistoryDisplay();
        }
        
        updateGameStatus();
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
    
    const containerHeight = chessboardElement.offsetHeight || 450;
    const chessboardWidth = containerHeight;
    const chessboardHeight = containerHeight;
    const squareSize = chessboardWidth / 8;
    
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
            square.className = 'chess-square';
            const pieceValue = boardData[row * 8 + col];
            
            // Set square position for absolute positioning
            square.style.width = `${squareSize}px`;
            square.style.height = `${squareSize}px`;
            square.style.position = 'absolute';
            square.style.left = `${visualCol * squareSize}px`;
            square.style.top = `${visualRow * squareSize}px`;
            
            // Apply checkerboard pattern
            const isLightSquare = (row + col) % 2 === 0;
            square.style.backgroundColor = isLightSquare ? '#f0d9b5' : '#b58863';
            
            let cursorStyle = "default";

            // Handle selected piece
            if (selectedPiece && selectedPiece.row === row && selectedPiece.col === col) {
                square.classList.add('selected');
            }
            
            // Handle possible moves
            if (possibleMoves.some(move => move.row === row && move.col === col)) {
                const isCapture = boardData[row * 8 + col] !== 0;
                square.classList.add(isCapture ? 'possible-capture' : 'possible-move');
                cursorStyle = "pointer";
            }

            square.style.cursor = cursorStyle;
            square.style.display = "flex";
            square.style.justifyContent = "center";
            square.style.alignItems = "center";

            if (pieceValue !== 0) {
                const pieceElement = document.createElement("img");
                pieceElement.src = getPieceImagePath(pieceValue);
                pieceElement.style.pointerEvents = 'none';
                pieceElement.style.width = '80%';
                pieceElement.style.height = '80%';
                pieceElement.style.objectFit = 'contain';
                pieceElement.className = 'chess-piece';
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
    
    // Client-side turn enforcement
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
                
                // animation, does this even work???? maybe not working due to the chessboard being 
                // rendered everytime we receive/send something to the websocket?
                const squares = chessboardElement.querySelectorAll('.chess-square');
                const targetSquare = Array.from(squares).find(sq => 
                    parseInt(sq.dataset.row) === movingPiece.endRow && 
                    parseInt(sq.dataset.col) === movingPiece.endCol
                );
                
                if (targetSquare) {
                    const piece = targetSquare.querySelector('.chess-piece');
                    if (piece) {
                        piece.classList.add('piece-moving');
                        setTimeout(() => piece.classList.remove('piece-moving'), 300);
                    }
                }
                
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
        // chessgame.load_fen(moveResult.newFEN);
        // checkGameEndConditions();

    } catch (error) {
        console.error("Failed to sync move with server:", error);
        alert(`A critical error occurred: ${error.message}. Please refresh.`);
    } finally {
        isProcessingMove = false;
        // drawChessboard();
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

function convertToAlgebraicNotation(moveStr, player) {
    // Simple conversion from coordinate notation to algebraic notation
    // This is a basic implementation - could be enhanced for castling, en passant, etc.
    const files = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];
    const pieces = {
        1: 'P', 2: 'N', 3: 'B', 4: 'R', 5: 'Q', 6: 'K', // White pieces
        9: 'p', 10: 'n', 11: 'b', 12: 'r', 13: 'q', 14: 'k' // Black pieces
    };
    
    if (moveStr.length < 4) return moveStr;
    
    const startCol = files.indexOf(moveStr[0]);
    const startRow = 8 - parseInt(moveStr[1]);
    const endCol = files.indexOf(moveStr[2]);
    const endRow = 8 - parseInt(moveStr[3]);
    
    const startPiece = chessgame.get_piece(startRow, startCol);
    const endPiece = chessgame.get_piece(endRow, endCol);
    const pieceSymbol = pieces[startPiece] || '';
    
    let notation = pieceSymbol;
    if (pieceSymbol === 'P' || pieceSymbol === 'p') {
        notation = ''; 
    }
    
    if (endPiece !== 0) {
        if (notation === '') {
            notation = moveStr[0];
        }
        notation += 'x';
    }
    
    notation += moveStr[2] + moveStr[3];
    
    if (chessgame.check()) {
        if (chessgame.checkmate()) {
            notation += '#';
        } else {
            notation += '+';
        }
    }
    
    return notation;
}

function addMoveToHistory(moveStr, player) {
    const notation = convertToAlgebraicNotation(moveStr, player);
    const moveEntry = {
        move: moveStr,
        notation: notation,
        player: player,
        timestamp: new Date(),
        moveNumber: moveNumber
    };
    
    moveHistory.push(moveEntry);
    
    if (player === 'black') {
        moveNumber++;
    }
    
    updateGameHistoryDisplay();
}
// TODO: this whole function should be moved to server.
function updateGameHistoryDisplay() {
    const historyElement = document.getElementById('game-history');
    if (!historyElement) return;
    
    const existingMoves = historyElement.querySelectorAll('.move-entry');
    existingMoves.forEach(move => move.remove());
    
    moveHistory.forEach((moveEntry, index) => {
        const moveDiv = document.createElement('div');
        moveDiv.className = 'move-entry';
        
        // Add move number for white moves
        if (moveEntry.player === 'white') {
            const moveNumberSpan = document.createElement('span');
            moveNumberSpan.className = 'move-number';
            moveNumberSpan.textContent = `${moveEntry.moveNumber}. `;
            moveDiv.appendChild(moveNumberSpan);
        }
        
        // Add notation
        const notationSpan = document.createElement('span');
        notationSpan.className = 'move-notation';
        notationSpan.textContent = moveEntry.notation;
        moveDiv.appendChild(notationSpan);
        
        // Highlight current move
        if (index === moveHistory.length - 1) {
            moveDiv.classList.add('current');
        }
        
        historyElement.appendChild(moveDiv);
    });
    
    historyElement.scrollTop = historyElement.scrollHeight;
}

function updateGameStatus() {
    const currentTurnElement = document.getElementById('current-turn');
    const playerColorElement = document.getElementById('player-color');
    const gameIdElement = document.getElementById('game-id');
    
    if (currentTurnElement && chessgame) {
        const currentTurnColor = chessgame.get_current_turn() === WHITE ? 'White' : 'Black';
        currentTurnElement.textContent = currentTurnColor;
    }
    
    if (playerColorElement && myPlayerColor) {
        playerColorElement.textContent = myPlayerColor.charAt(0).toUpperCase() + myPlayerColor.slice(1);
        playerColorElement.className = `status-value ${myPlayerColor}`;
    }
    
    if (gameIdElement && currentGameID) {
        gameIdElement.textContent = currentGameID;
    }
}


initializePage();