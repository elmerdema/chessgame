use wasm_bindgen::prelude::*;

use crate::constants::{
    W_PAWN, W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING,
    B_PAWN, B_ROOK, B_KNIGHT, B_BISHOP, B_QUEEN, B_KING,
    WHITE,
};
use crate::ChessGame;

#[wasm_bindgen]
impl ChessGame {
    #[wasm_bindgen(constructor)]
    pub fn new() -> ChessGame {
        let mut board = vec![vec![0; 8]; 8];
        // Black pieces (Rank 8 -> board index 0)
        board[0] = vec![B_ROOK,B_KNIGHT, B_BISHOP, B_QUEEN, B_KING, B_BISHOP, B_KNIGHT, B_ROOK];
        // Black Pawns (Rank 7 -> board index 1)
        board[1] = vec![B_PAWN; 8];
        board[6] = vec![ W_PAWN; 8];
        board[7] = vec![W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING, W_BISHOP, W_KNIGHT, W_ROOK];

        ChessGame {board, current_turn: WHITE, white_can_castle_kingside: true, white_can_castle_queenside: true, black_can_castle_kingside: true, black_can_castle_queenside: true, en_passant_target: None}
    }

    pub fn get_board_json(&self) -> String {
        match serde_json::to_string(&self.board) {
            Ok(json) => json,
            // Todo:Consider using wasm_bindgen::JsValue for errors???
            Err(e) => format!("{{\"error\": \"{}\"}}", e),
        }
    }

    // Rreturn a flattened 1D array which JS can handle
    pub fn get_board(&self) -> Vec<i32> {
        self.board.iter().flat_map(|row| row.iter().cloned()).collect()
    }

    pub fn get_board_width(&self) -> usize {
        // Assumes a square board or at least one row if not empty
        self.board.get(0).map_or(0, |row| row.len())
    }

    pub fn get_board_height(&self) -> usize {
        self.board.len()
    }

    pub fn get_piece(&self, x: usize, y: usize) -> i32 {
        // Check bounds using usize which cannot be negative
        if x >= self.get_board_height() || y >= self.get_board_width() {
            return 0;
        }
        self.board[x][y]
    }

    // Internal helper to check if coords are on the board
    pub(crate) fn is_on_board(x: i32, y: i32) -> bool {
        x >= 0 && x < 8 && y >= 0 && y < 8
    }
}
