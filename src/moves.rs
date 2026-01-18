use wasm_bindgen::prelude::*;
use wasm_bindgen::JsValue;

use crate::constants::{
    W_KING, B_KING,
    WHITE, BLACK,
};
use crate::utils;
use crate::ChessGame;

#[wasm_bindgen]
impl ChessGame {
    #[wasm_bindgen]
    pub fn make_move(&mut self, start_x: usize, start_y: usize, end_x: usize, end_y: usize) -> Result<Option<Vec<usize>>, JsValue> {

        let (sx, sy, ex, ey) = (start_x as i32, start_y as i32, end_x as i32, end_y as i32);

        // Get piece info *before* the move, for validation and then for promotion check
        let piece_at_start = self.get_piece(start_x, start_y);
        // let piece_type_at_start = utils::get_piece_type(piece_at_start);
        let piece_color_at_start = utils::get_piece_color(piece_at_start);

        if !self.is_valid_move(sx, sy, ex, ey) {
            let error_msg = if piece_at_start == 0 {
                "No piece at starting square.".to_string()
            } else if piece_color_at_start != self.current_turn {
                format!("Not {}'s turn.", if self.current_turn == WHITE { "White" } else { "Black" })
            } else {
                format!("Invalid move for piece from ({},{}) to ({},{}). Check move shape, path, or if king would be in check.", sx, sy, ex, ey)
            };
            return Err(JsValue::from_str(&error_msg));
        }

        // Perform the move internally. This *doesn't* promote the pawn or switch turns.
        self.make_move_internal(sx, sy, ex, ey, false)
            .map_err(|e| JsValue::from_str(&format!("Internal move error: {}", e)))?;

        // After `make_move_internal` completes, the piece is at `end_x, end_y`.
        // Check for promotion after the move has been applied to the board.
        let promotion_coords: Option<Vec<usize>> = {
            let moved_piece = self.get_piece(end_x, end_y);
            let moved_piece_type = utils::get_piece_type(moved_piece);
            let moved_piece_color = utils::get_piece_color(moved_piece);

            let is_pawn_promotion_rank = (moved_piece_color == WHITE && end_x == 0) ||
                                         (moved_piece_color == BLACK && end_x == 7);

            if moved_piece_type == 1 && is_pawn_promotion_rank {
                Some(vec![end_x, end_y])
            } else {
                None
            }
        };

        // If a promotion is pending, the turn is NOT switched yet.
        // It will be switched after the promotion choice is made and applied by `promote_pawn`.
        if promotion_coords.is_none() {
            // If no promotion, then switch turns as usual
            self.current_turn = if self.current_turn == WHITE { BLACK } else { WHITE };
        }

        // Return the promotion coordinates (if any) to JavaScript
        Ok(promotion_coords)
    }

    // Internal helper to perform the move mechanics and state updates.
    // `is_simulation` prevents recursive state updates during validation check.
    pub(crate) fn make_move_internal(&mut self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, is_simulation: bool) -> Result<(), String> {
        let piece = self.board[start_x as usize][start_y as usize];
        let piece_type = utils::get_piece_type(piece);
        let piece_color = utils::get_piece_color(piece); // Keep this for castling rights updates

        // --- Handle special captures/moves before moving the piece ---
        // 1. Handle En Passant Capture (remove the passed pawn)
        if piece_type == 1 && self.is_en_passant_move(start_x, start_y, end_x, end_y) {
            let captured_pawn_x = start_x; // Captured pawn is on the start rank
            let captured_pawn_y = end_y;   // Captured pawn is on the destination file
            self.board[captured_pawn_x as usize][captured_pawn_y as usize] = 0;
        }

        // 2. Move the piece
        self.board[end_x as usize][end_y as usize] = piece;
        self.board[start_x as usize][start_y as usize] = 0;

        // 3. Handle Castling (move the rook)
        if piece_type == 6 && (end_y - start_y).abs() == 2 {
            let (rook_start_y, rook_end_y) = if end_y > start_y {
                (7, 5) // Kingside: H-file rook to F-file
            } else {
                (0, 3) // Queenside: A-file rook to D-file
            };
            let rook = self.board[start_x as usize][rook_start_y];
            self.board[start_x as usize][rook_end_y] = rook;
            self.board[start_x as usize][rook_start_y] = 0;
        }

        // --- Post-Move State Update (only if not a simulation) ---
        if !is_simulation {
            // Update Castling Rights
            // If King moved
            if piece_type == 6 {
                if piece_color == WHITE {
                    self.white_can_castle_kingside = false;
                    self.white_can_castle_queenside = false;
                } else {
                    self.black_can_castle_kingside = false;
                    self.black_can_castle_queenside = false;
                }
            }
            // If Rook moved (from its starting square)
            if piece_type == 2 {
                if piece_color == WHITE {
                    if start_x == 7 && start_y == 0 { self.white_can_castle_queenside = false; }
                    if start_x == 7 && start_y == 7 { self.white_can_castle_kingside = false; }
                } else { // Black
                    if start_x == 0 && start_y == 0 { self.black_can_castle_queenside = false; }
                    if start_x == 0 && start_y == 7 { self.black_can_castle_kingside = false; }
                }
            }

            // Set potential En Passant target for the *next* turn
             if piece_type == 1 && (end_x - start_x).abs() == 2 {
                 let target_rank = (start_x + end_x) / 2; // Rank behind the pawn
                 self.en_passant_target = Some((target_rank as usize, start_y as usize));
             } else {
                 self.en_passant_target = None; // Clear en passant target if not a double pawn push???
             }

        }

        Ok(())
    }

    pub fn get_moves(&self, x:usize, y:usize) -> Vec<usize> {
        let mut moves= Vec::new();

        if x > 7 || y > 7 {
            return moves;
        }

        let piece= self.get_piece(x, y);
        if piece == 0 {
            return moves;
        }
        if utils::get_piece_color(piece) != self.current_turn {
            return moves;
        }


        for i in 0..8{
            for j in 0..8{
                if self.is_valid_move(x as i32, y as i32, i as i32, j as i32) {
                    moves.push(i);
                    moves.push(j);
                }
            }
        }
        moves
    }

    
    #[wasm_bindgen]
    pub fn get_king_position(&self) -> Vec<i32> {
        let king_piece = if self.current_turn == WHITE { W_KING } else { B_KING };

        for r in 0..8 {
            for c in 0..8 {
                if self.board[r][c] == king_piece {
                    return vec![r as i32, c as i32];
                }
            }
        }
        // Should not happen in a valid game, but return an empty vector if king not found
        Vec::new() 
    }

    // Checkmate logic needs to consider the current player's turn.
    // If checkmate is true, it means the *current player* has no legal moves.
    pub fn checkmate(&self) -> bool {
        // If the current player is not in check, it cannot be checkmate.
        if !self.is_check(self.current_turn) {
            return false;
        }
        // If in check, check if any piece of the current turn can make a valid move.
        // If no moves are found, it's checkmate.
        for r in 0..8 {
            for c in 0..8 {
                let temp_piece = self.get_piece(r, c);
                if temp_piece != 0 && utils::get_piece_color(temp_piece) == self.current_turn {
                    // If any piece can make a valid move, it's not checkmate
                    if self.get_moves(r, c).len() > 0 {
                        return false;
                    }
                }
            }
        }
        true // No legal moves found for the player in check -> checkmate
    }

    pub fn get_current_turn(&self) -> i32 {
        self.current_turn
    }
    pub fn change_turn(&mut self) {
        self.current_turn = if self.current_turn == WHITE { BLACK } else { WHITE };
    }
    pub fn check(&self) -> bool {
        self.is_check(self.current_turn)
    }

    pub fn can_castle_kingside(&self) -> bool {
        if self.current_turn == WHITE {
            self.white_can_castle_kingside
        }
        else {
            self.black_can_castle_kingside
        }
    }

    pub fn is_stalemate(&self) -> bool {
        // If the current player is in check, it cannot be stalemate.
        if self.is_check(self.current_turn) {
            return false;
        }
        // If not in check, check if any piece of the current turn can make a valid move.
        // If no moves are found, it's stalemate.
        for r in 0..8 {
            for c in 0..8 {
                let piece = self.get_piece(r, c);
                if piece != 0 && utils::get_piece_color(piece) == self.current_turn {
                    // If any piece can make a valid move, it's not stalemate
                    if self.get_moves(r, c).len() > 0 {
                        return false;
                    }
                }
            }
        }
        true // Not in check, but no legal moves -> stalemate
    }

    #[wasm_bindgen]
    pub fn promote_pawn(&mut self, x: usize, y: usize, new_piece: i32) -> Result<(), JsValue> {
        if x > 7 || y > 7 {
            return Err(JsValue::from_str("Coordinates out of bounds"));
        }
        let piece = self.get_piece(x, y);
        let piece_type = utils::get_piece_type(piece);
        let piece_color = utils::get_piece_color(piece);

        if piece == 0 || piece_type != 1 {
            return Err(JsValue::from_str("No pawn at specified coordinates for promotion"));
        }
        
        // Ensure the pawn is on the correct promotion rank
        let promotion_rank = if piece_color == WHITE { 0 } else { 7 };
        if x != promotion_rank {
            return Err(JsValue::from_str("Pawn is not on the promotion rank"));
        }

        // Basic validation for new_piece (ensure it's a valid promotion piece of the correct color)
        let new_piece_type = utils::get_piece_type(new_piece);
        let new_piece_color = utils::get_piece_color(new_piece);
        
        let is_valid_promotion_piece = match new_piece_type {
            2 | 3 | 4 | 5 => true, // Rook, Knight, Bishop, Queen
            _ => false,
        };

        if !is_valid_promotion_piece || new_piece_color != piece_color {
            return Err(JsValue::from_str("Invalid piece for promotion"));
        }

        // Replace the pawn with the new piece
        self.board[x][y] = new_piece;
        Ok(())
    }
}
