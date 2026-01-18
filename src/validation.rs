use wasm_bindgen::prelude::*;

use crate::constants::{
    W_KING, B_KING,
    WHITE, BLACK,
};
use crate::utils;
use crate::ChessGame;

#[wasm_bindgen]
impl ChessGame {
    pub fn is_valid_move(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
        utils::log(&format!("Checking move from ({},{}) to ({},{})", start_x, start_y, end_x, end_y));
        
        // 1. Basic Bounds & Same Square
        if !Self::is_on_board(start_x, start_y) || !Self::is_on_board(end_x, end_y) { 
            utils::log("Move failed: out of bounds");
            return false; 
        }
        if start_x == end_x && start_y == end_y { 
            utils::log("Move failed: same square");
            return false; 
        }

        // 2. Get Piece Info & Check Turn
        let piece = self.get_piece(start_x as usize, start_y as usize);
        if piece == 0 { 
            utils::log("Move failed: no piece at start");
            return false; 
        }
        
        let piece_color = utils::get_piece_color(piece);
        if piece_color != self.current_turn { 
            utils::log(&format!("Move failed: wrong turn. Piece color: {}, Current turn: {}", piece_color, self.current_turn));
            return false; 
        }

        let piece_type = utils::get_piece_type(piece);
        utils::log(&format!("Moving piece type: {}, color: {}", piece_type, piece_color));

        // 3. Get Destination Info
        let ending_piece = self.get_piece(end_x as usize, end_y as usize);
        let ending_piece_color = utils::get_piece_color(ending_piece);

        // 4. Check Target Square (Capture Rule)
        if ending_piece != 0 && ending_piece_color == piece_color { return false; } // Cannot capture friendly piece

        // 5. Validate Move Shape, Obstructions, and Special Moves
        let is_basic_move_valid = match piece_type {
            1 => self.is_valid_pawn_move_detailed(start_x, start_y, end_x, end_y, piece_color, ending_piece, ending_piece_color),
            2 => {
                utils::is_valid_rook_move(start_x, start_y, end_x, end_y) &&
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            3 => utils::is_valid_knight_move(start_x, start_y, end_x, end_y),
            4 => {
                utils::is_valid_bishop_move(start_x, start_y, end_x, end_y) &&
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            5 => {
                utils::is_valid_queen_move(start_x, start_y, end_x, end_y) &&
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            6 => self.is_valid_king_move_detailed(start_x, start_y, end_x, end_y, piece_color),
            _ => false // Unknown piece type just in case?
        };
        // If the basic shape/rules aren't valid, the move is impossible.
        if !is_basic_move_valid {
            return false;
        }

        // 6. Check if Move Puts Own King in Check (Simulation)
        let mut temp_game = self.clone();

        //  simulate the move, It won't update state or turn.
        if let Err(_) = temp_game.make_move_internal(start_x, start_y, end_x, end_y, true) {
            // this should ideally not happen if is_basic_move_valid is true, but just to be sure
            return false; 
        }

        if temp_game.is_check(piece_color) {
            utils::log("Move failed: leaves king in check");
            return false; // Move is invalid, the king in check
        }
        true
    }

    pub(crate) fn is_en_passant_move(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
        if let Some(target_square) = self.en_passant_target {
            let piece = self.get_piece(start_x as usize, start_y as usize);
             // Check if it's a pawn moving diagonally to the en passant target square
             utils::get_piece_type(piece) == 1 &&
             (end_x as usize, end_y as usize) == target_square &&
             (start_y - end_y).abs() == 1
        } else {
            false
        }
    }
    fn is_valid_pawn_move_detailed(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, piece_color: i32, ending_piece: i32, ending_piece_color: i32) -> bool {
        let is_white = piece_color == WHITE;
        let dx = end_x - start_x;
        let dy = end_y - start_y;

        // Standard Forward Moves
        if dy == 0 {
            if ending_piece != 0 { 
                utils::log("Pawn cannot move forward onto a piece");
                return false; 
            }
            
            let expected_dx = if is_white { -1 } else { 1 };
            let start_rank = if is_white { 6 } else { 1 };

            utils::log(&format!("Expected dx: {}, start_rank: {}, actual start_x: {}", expected_dx, start_rank, start_x));

            // Single step forward
            if dx == expected_dx { 
                utils::log("Valid single pawn move");
                return true; 
            }

            // Double step forward (only from starting rank)
            if dx == expected_dx * 2 && start_x == start_rank {
                // Check if path is clear
                let middle_x = start_x + expected_dx;
                if self.get_piece(middle_x as usize, start_y as usize) != 0 {
                    utils::log("Pawn double move blocked");
                    return false;
                }
                utils::log("Valid double pawn move");
                return true;
            }
            utils::log(&format!("Invalid pawn forward move: dx={}, expected={}, start_x={}, start_rank={}", dx, expected_dx, start_x, start_rank));
            return false;
        }
        // Diagonal Moves (Capture or En Passant)
        else if dy.abs() == 1 {
            let expected_dx = if is_white { -1 } else { 1 };
            if dx != expected_dx { 
                utils::log("Invalid pawn diagonal direction");
                return false; 
            }

            if ending_piece != 0 && ending_piece_color != piece_color {
                utils::log("Valid pawn capture");
                return true;
            }
            if ending_piece == 0 && self.is_en_passant_move(start_x, start_y, end_x, end_y) {
                utils::log("Valid en passant");
                return true;
            }
            utils::log("Invalid pawn diagonal move");
            return false;
        }
        else {
            utils::log("Invalid pawn move shape");
            false
        }
    }

    fn is_valid_king_move_detailed(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, piece_color: i32) -> bool {
        let dx = end_x - start_x;
        let dy = end_y - start_y;

        
        if dx.abs() <= 1 && dy.abs() <= 1 {
            return true;
        }

        // Must be moving exactly 2 squares horizontally, and no vertical movement
        if dx == 0 && dy.abs() == 2 {
            // Cannot castle if currently in check
            if self.is_check(piece_color) { return false; }

            let opponent_color = if piece_color == WHITE { BLACK } else { WHITE };

            // Kingside Castling (O-O)
            if end_y > start_y { // e.g., e1 -> g1 (y=4 -> y=6) or e8 -> g8 (y=4 -> y=6)
                let can_castle = if piece_color == WHITE { self.white_can_castle_kingside } else { self.black_can_castle_kingside };
                if !can_castle { return false; }

                // Check path clear between king and rook (f1/f8, g1/g8)
                if self.get_piece(start_x as usize, (start_y + 1) as usize) != 0 || // f1/f8
                   self.get_piece(start_x as usize, (start_y + 2) as usize) != 0    // g1/g8 (king destination)
                { return false; }

                // Check squares king passes through/lands on are not attacked
                if self.is_square_attacked(start_x, start_y + 1, opponent_color) || // f1/f8
                   self.is_square_attacked(start_x, start_y + 2, opponent_color)    // g1/g8
                { return false; }

                return true;
            }
            // Queenside Castling (O-O-O)
            else { // e.g., e1 -> c1 (y=4 -> y=2) or e8 -> c8 (y=4 -> y=2)
                let can_castle = if piece_color == WHITE { self.white_can_castle_queenside } else { self.black_can_castle_queenside };
                if !can_castle { return false; }

                // Check path clear between king and rook (d1/d8, c1/c8, b1/b8)
                 // Note: b1/b8 doesn't need to be empty for castling itself, only for rook path!
                if self.get_piece(start_x as usize, (start_y - 1) as usize) != 0 || // d1/d8
                   self.get_piece(start_x as usize, (start_y - 2) as usize) != 0 || // c1/c8 (king destination)
                   self.get_piece(start_x as usize, (start_y - 3) as usize) != 0    // b1/b8
                { return false; }

                // Check squares king passes through/lands on are not attacked
                if self.is_square_attacked(start_x, start_y - 1, opponent_color) || // d1/d8
                   self.is_square_attacked(start_x, start_y - 2, opponent_color)    // c1/c8
                { return false; }

                return true; // Valid Queenside Castle attempt
            }
        }

        // If not a standard move or castling, it's invalid
        false
    }


    /// Helper function to check if the path is clear for Rook, Bishop, Queen moves.
    fn is_path_clear(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
        let dx = end_x - start_x;
        let dy = end_y - start_y;

        let step_x = dx.signum(); // -1, 0, or 1
        let step_y = dy.signum(); // -1, 0, or 1

        let mut current_x = start_x + step_x;
        let mut current_y = start_y + step_y;

        // Iterate until one step before the end square
        while current_x != end_x || current_y != end_y {
            if !Self::is_on_board(current_x, current_y) {
                // This shouldn't happen if start/end are on board and move shape is valid, but safety check
                return false;
            }
            if self.get_piece(current_x as usize, current_y as usize) != 0 {
                return false; // Path is blocked
            }
            current_x += step_x;
            current_y += step_y;
        }

        true // Path is clear
    }

    /// Checks if the king of the specified color is currently in check.
    /// color: 1 for White, 2 for Black
    pub fn is_check(&self, color: i32) -> bool {
        // 1. Find the King
        let king_piece = if color == WHITE { W_KING } else { B_KING };
        let mut king_x = -1;
        let mut king_y = -1;

        for r in 0..8 {
            for c in 0..8 {
                if self.board[r][c] == king_piece {
                    king_x = r as i32;
                    king_y = c as i32;
                    break;
                }
            }
            if king_x != -1 { break; }
        }

        if king_x == -1 {
             utils::log(&format!("King for color {} not found!", color));
            return false; // Improve error handling?
        }

        // 2. Check if the king's square is attacked by the opponent
        let opponent_color = if color == WHITE { BLACK } else { WHITE };
        self.is_square_attacked(king_x, king_y, opponent_color)
    }
    fn is_square_attacked(&self, target_x: i32, target_y: i32, attacker_color: i32) -> bool {
        if !Self::is_on_board(target_x, target_y) {
            return false; // theoricallyshould not happen if called correctly
        }

        for r in 0..8 {
            for c in 0..8 {
                let piece = self.board[r][c];
                if piece != 0 && utils::get_piece_color(piece) == attacker_color {
                    // Check if this opponent piece can attack the target square
                    if self.is_valid_move_shape_and_obstruction(r as i32, c as i32, target_x, target_y) {
                        return true;
                    }
                }
            }
        }
        false
    }

    fn is_valid_move_shape_and_obstruction(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {

        // 1. Bounds and Same Square (already checked by caller usually, but safe)
        if !Self::is_on_board(start_x, start_y) || !Self::is_on_board(end_x, end_y) { return false; }
        if start_x == end_x && start_y == end_y { return false; }

        let piece = self.get_piece(start_x as usize, start_y as usize);
        if piece == 0 { return false; }
        let piece_color = utils::get_piece_color(piece);
        let piece_type = utils::get_piece_type(piece);

        let ending_piece = self.get_piece(end_x as usize, end_y as usize);
        let ending_piece_color = utils::get_piece_color(ending_piece);

        // (Cannot capture friendly piece)
        if ending_piece != 0 && ending_piece_color == piece_color { return false; }

        match piece_type {
            1 => { // Pawn
                let is_white_attacker = piece_color == WHITE;
                let dx = end_x - start_x;
                let dy = end_y - start_y;

                if dy.abs() == 1 { // Diagonal Capture
                    let expected_dx_for_attack = if is_white_attacker {-1} else {1}; //must be 1 square forward
                    return dx == expected_dx_for_attack;
                }
                //If not a one-step diagonal move, the pawn is not attacking this square.
                return false;
            },
            2 => {
                if !utils::is_valid_rook_move(start_x, start_y, end_x, end_y) { return false; }
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            3 => {
                utils::is_valid_knight_move(start_x, start_y, end_x, end_y)
            },
            4 => {
                if !utils::is_valid_bishop_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            5 => {
                if !utils::is_valid_queen_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            6 => {
                // For attack checks, we only care about the one-square move shape.
                // Castling is handled separately in the main is_valid_move.
                utils::is_valid_king_move(start_x, start_y, end_x, end_y)
            },
            _ => false
        }
    }
}
