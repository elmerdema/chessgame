import { getPieceImagePath } from "./index.js";


let _chessgame = null;
let _promotionConstants = {};
let promotionInfo = null; //  stores { startX, startY, endX, endY }
let onPromotionCompletedCallback = null;
const TRANSITION_DURATION = 300; 

const promotionOverlay = document.getElementById("promotion-overlay");
const promotionDialog = document.getElementById("promotion-dialog");
const promotionChoicesContainer = document.getElementById("promotion-choices");

function getPromotionElements() {
    return { promotionOverlay, promotionDialog, promotionChoicesContainer };
}

export function setPromotionGameAndConstants(gameInstance, constants) {
    _chessgame = gameInstance;
    _promotionConstants = constants;
}

export function setOnPromotionCompleted(callback) {
    onPromotionCompletedCallback = callback;
}

export async function onPromotionChoice(event) {
    const chosenPieceValue = parseInt(event.currentTarget.dataset.pieceValue);

    if (!promotionInfo) {
        console.error("No pawn promotion pending, but choice was made!");
        hidePromotionDialog();
        return;
    }
    
    // Deconstruct for clarity
    const { startX, startY, endX, endY } = promotionInfo;

    try {
        if (!_chessgame) throw new Error("Chess game instance not set.");
        
        await _chessgame.promote_pawn(endX, endY, chosenPieceValue);
        _chessgame.change_turn();

        hidePromotionDialog();

        if (onPromotionCompletedCallback) {
            // Convert chosen piece to a character for the server
            let promotionChar = '';
            const pieceType = (chosenPieceValue - 1) % 6 + 1;
            switch (pieceType) {
                case 5: promotionChar = 'q'; break; // Queen
                case 2: promotionChar = 'r'; break; // Rook
                case 4: promotionChar = 'b'; break; // Bishop
                case 3: promotionChar = 'n'; break; // Knight
            }
            // callback with all info needed to sync with the server
            onPromotionCompletedCallback(startX, startY, endX, endY, promotionChar);
        }

    } catch (error) {
        console.error("Error promoting pawn:", error);
        alert(`Promotion failed: ${error.message || error}`);
        hidePromotionDialog();
        // Redraw to revert to pre-promotion state
        if (onPromotionCompletedCallback) onPromotionCompletedCallback();
    }
}

export function hidePromotionDialog() {
    const { promotionOverlay, promotionDialog } = getPromotionElements();
    promotionOverlay.classList.remove("visible");
    promotionDialog.classList.remove("visible");
    promotionInfo = null;

    setTimeout(() => {
        promotionOverlay.classList.add("hidden");
        promotionDialog.classList.add("hidden");
        promotionInfo = null;
    }, TRANSITION_DURATION);
}
export function showPromotionDialog(startX, startY, endX, endY) {
    console.log("Showing promotion dialog");
    const { promotionOverlay, promotionDialog, promotionChoicesContainer } = getPromotionElements();
    promotionChoicesContainer.innerHTML = "";
    
    promotionInfo = { startX, startY, endX, endY };

    const playerColor = _chessgame.get_current_turn();

    const promotionPieces = [
        { type: "Queen", value: playerColor === _promotionConstants.WHITE ? _promotionConstants.W_QUEEN : _promotionConstants.B_QUEEN },
        { type: "Rook", value: playerColor === _promotionConstants.WHITE ? _promotionConstants.W_ROOK : _promotionConstants.B_ROOK },
        { type: "Knight", value: playerColor === _promotionConstants.WHITE ? _promotionConstants.W_KNIGHT : _promotionConstants.B_KNIGHT },
        { type: "Bishop", value: playerColor === _promotionConstants.WHITE ? _promotionConstants.W_BISHOP : _promotionConstants.B_BISHOP },
    ];

    promotionPieces.forEach(piece => {
        const choiceDiv = document.createElement("div");
        choiceDiv.classList.add("promotion-choice");
        const pieceImage = document.createElement("img");
        pieceImage.src = getPieceImagePath(piece.value);
        choiceDiv.appendChild(pieceImage);
        choiceDiv.dataset.pieceValue = piece.value;
        choiceDiv.addEventListener("click", onPromotionChoice);
        promotionChoicesContainer.appendChild(choiceDiv);
    });

    promotionOverlay.classList.remove("hidden");
    promotionDialog.classList.remove("hidden");
    
    setTimeout(() => {
        promotionOverlay.classList.add("visible");
        promotionDialog.classList.add("visible");
    }, 10);
}