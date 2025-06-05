import {
    getPieceImagePath,
    WHITE, BLACK,
    W_QUEEN, W_ROOK, W_KNIGHT, W_BISHOP,
    B_QUEEN, B_ROOK, B_KNIGHT, B_BISHOP
} from "./index.js";

let _chessgame = null;
let _promotionConstants = {};
let promotionPawnCoords = null;

let onPromotionCompletedCallback = null;

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

// Allows index.js to set the callback for post-promotion actions
export function setOnPromotionCompleted(callback) {
    onPromotionCompletedCallback = callback;
}

export async function onPromotionChoice(event) {
    const chosenPieceValue = parseInt(event.currentTarget.dataset.pieceValue);
    const chosenPieceType = event.currentTarget.dataset.pieceType;

    if (!promotionPawnCoords) {
        console.error("No pawn promotion pending, but choice was made!");
        hidePromotionDialog();
        return;
    }

    const { x, y } = promotionPawnCoords;

    try {
        // Ensure chessgame instance is available before trying to use it
        if (!_chessgame) {
            throw new Error("Chess game instance not set in promotion module.");
        }
        await _chessgame.promote_pawn(x, y, chosenPieceValue);
        console.log(`Pawn at (${x},${y}) promoted to ${chosenPieceType}.`);

        hidePromotionDialog();

        // Call the callback to signal index.js to redraw and check end conditions
        if (onPromotionCompletedCallback) {
            onPromotionCompletedCallback();
        }

        // REMOVE THIS LINE: chessgame.change_turn(); // Already handled by make_move

    } catch (error) {
        console.error("Error promoting pawn:", error);
        alert(`Promotion failed: ${error.message || error}`);
        hidePromotionDialog();
        // Also call callback on error to redraw the board state, if needed
        if (onPromotionCompletedCallback) {
            onPromotionCompletedCallback();
        }
    }
}

export function hidePromotionDialog() {
    const { promotionOverlay, promotionDialog } = getPromotionElements();
    promotionOverlay.classList.remove("visible");
    promotionOverlay.classList.add("hidden");
    promotionDialog.classList.remove("visible");
    promotionDialog.classList.add("hidden");
    promotionPawnCoords = null; // Clear the stored coordinates
}


export function showPromotionDialog(pawnX, pawnY) {
    const { promotionOverlay, promotionDialog, promotionChoicesContainer } = getPromotionElements();
    promotionChoicesContainer.innerHTML = "";


    // Store the pawn coordinates internally for use by onPromotionChoice
    promotionPawnCoords = { x: pawnX, y: pawnY };

    // Use the stored chessgame instance to get the current turn
    const playerColor = _chessgame.get_current_turn();

    const promotionPieces = [
        // Use constants from _promotionConstants
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