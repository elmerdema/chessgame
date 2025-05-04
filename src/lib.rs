mod utils;
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsValue;
use std::convert::TryInto;

//wasm-pack build
//cd www
// npm run start

const W_PAWN: i32 = 1;
const W_ROOK: i32 = 2;
const W_KNIGHT: i32 = 3;
const W_BISHOP: i32 = 4;
const W_QUEEN: i32 = 5;
const W_KING: i32 = 6;
const B_PAWN: i32 = 7;
const B_ROOK: i32 = 8;
const B_KNIGHT: i32 = 9;
const B_BISHOP: i32 = 10;
const B_QUEEN: i32 = 11;
const B_KING: i32 = 12;

const WHITE: i32 = 1;
const BLACK: i32 = 2;

#[wasm_bindgen]
#[derive(Debug, Clone)]
pub struct ChessGame {
    board: Vec<Vec<i32>>,
    current_turn: i32,
    white_can_castle_kingside: bool,
    white_can_castle_queenside: bool,
    black_can_castle_kingside: bool,
    black_can_castle_queenside: bool,

    en_passant_target: Option<(usize,usize)>
}

#[wasm_bindgen]
impl ChessGame {
    #[wasm_bindgen(constructor)]
    pub fn new() -> ChessGame {
        let mut board = vec![vec![0; 8]; 8];


        // Black pieces (Rank 8 -> board index 0)
        board[0] = vec![B_ROOK,B_KNIGHT, B_BISHOP, B_QUEEN, B_KING, B_BISHOP, B_BISHOP, B_ROOK];
        // Black Pawns (Rank 7 -> board index 1)
        board[1] = vec![B_PAWN; 8];    // B_Pawn * 8
        board[6] = vec![ W_PAWN; 8];    // W_Pawn * 8
        board[7] = vec![W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING, W_BISHOP, W_KNIGHT, W_ROOK];

        ChessGame {board, current_turn: WHITE, white_can_castle_kingside: true, white_can_castle_queenside: true, black_can_castle_kingside: true, black_can_castle_queenside: true, en_passant_target: None}
    }

    // Return a serialized JSON
    pub fn get_board_json(&self) -> String {
        match serde_json::to_string(&self.board) {
            Ok(json) => json,
            // Todo:Consider using wasm_bindgen::JsValue for errors???
            Err(e) => format!("{{\"error\": \"{}\"}}", e),
        }
    }

    // Return a flattened 1D array which JS can handle
    pub fn get_board(&self) -> Vec<i32> {
        self.board.iter().flat_map(|row| row.iter().cloned()).collect()
    }

    //  Return board dimensions separately
    pub fn get_board_width(&self) -> usize {
        // Assumes a square board or at least one row if not empty
        self.board.get(0).map_or(0, |row| row.len())
    }

    pub fn get_board_height(&self) -> usize {
        self.board.len()
    }

    // Get a specific piece using usize for indexing internally
    pub fn get_piece(&self, x: usize, y: usize) -> i32 {
        // Check bounds using usize which cannot be negative
        if x >= self.get_board_height() || y >= self.get_board_width() {
            return 0; // Or handle as an error/Option<i32>
        }
        self.board[x][y]
    }

    // Internal helper to check if coords are on the board
    fn is_on_board(x: i32, y: i32) -> bool {
        x >= 0 && x < 8 && y >= 0 && y < 8
    }

    pub fn move_piece(&mut self, start_x: usize, start_y: usize, end_x: usize, end_y: usize) -> bool {
         // Basic bounds check (using usize which handles >= 0 implicitly)
        if start_x >= 8 || start_y >= 8 || end_x >= 8 || end_y >= 8 {
             utils::log("Move coordinates out of bounds."); // Requires `#[wasm_bindgen(module = "/utils.js")] extern "C" { fn log(s: &str); }` or similar
            return false;
        }

        let piece = self.board[start_x][start_y];
        if piece == 0 {
            utils::log("No piece at starting square.");
            return false; // Cannot move an empty square
        }

        self.board[end_x][end_y] = piece;
        self.board[start_x][start_y] = 0;
        true
    }

    pub fn is_valid_move(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
        // 1. Basic Bounds & Same Square
        if !Self::is_on_board(start_x, start_y) || !Self::is_on_board(end_x, end_y) { return false; }
        if start_x == end_x && start_y == end_y { return false; }

        // 2. Get Piece Info & Check Turn
        let piece = self.get_piece(start_x as usize, start_y as usize);
        if piece == 0 { return false; } // No piece to move
        let piece_color = utils::get_piece_color(piece);
        if piece_color != self.current_turn { return false; } // Not this player's turn

        let piece_type = utils::get_piece_type(piece);

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
            3 => utils::is_valid_knight_move(start_x, start_y, end_x, end_y), // Jumps over pieces
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
        // Create a temporary copy of the game state
        let mut temp_game = self.clone();
        let captured_piece = temp_game.board[end_x as usize][end_y as usize]; // Store potential captured piece (for revert, though not strictly needed here)
        temp_game.board[end_x as usize][end_y as usize] = piece;
        temp_game.board[start_x as usize][start_y as usize] = 0;

        // Handle en passant capture removal in simulation
        if piece_type == 1 && self.is_en_passant_move(start_x, start_y, end_x, end_y) {
             let captured_pawn_x = start_x; // En passant captured pawn is on same rank as start
             let captured_pawn_y = end_y;
             temp_game.board[captured_pawn_x as usize][captured_pawn_y as usize] = 0;
        }
        // Handle castling rook movement in simulation
        if piece_type == 6 && (end_y - start_y).abs() == 2 {
            let (rook_start_y, rook_end_y) = if end_y > start_y { (7, 5) } else { (0, 3) }; // Kingside / Queenside
            let rook = temp_game.board[start_x as usize][rook_start_y];
            temp_game.board[start_x as usize][rook_end_y] = rook;
            temp_game.board[start_x as usize][rook_start_y] = 0;
        }


        // Now, check if the king of the player who moved is in check in the tewmp state
        if temp_game.is_check(piece_color) {
            return false; // Move is invalid because it leaves the king in check
        }
        true
    }

    // Helper to determine if a move coordinates constitute an en passant capture
    fn is_en_passant_move(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
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

    // Extracted detailed pawn move logic for clarity
    fn is_valid_pawn_move_detailed(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, piece_color: i32, ending_piece: i32, ending_piece_color: i32) -> bool {
        let is_white = piece_color == WHITE;
        let dx = end_x - start_x; //delta x and y
        let dy = end_y - start_y; 

        // Standard Forward Moves
        if dy == 0 {
            if ending_piece != 0 { return false; } // Cannot move forward onto a piece
            let expected_dx = if is_white { -1 } else { 1 };
            let expected_dx_double = if is_white { -2 } else { 2 };
            let start_rank = if is_white { 6 } else { 1 }; // Rank 2 for white, Rank 7 for black

            // Single step forward
            if dx == expected_dx { return true; }

            // Double step forward (only from starting rank)
            if dx == expected_dx_double && start_x == start_rank {
                // Check obstruction
                let middle_x = start_x + expected_dx;
                if self.get_piece(middle_x as usize, start_y as usize) != 0 {
                    return false; // Path is cockblocked
                }
                return true; // Valid double move
            }
            return false; // Invalid forward move shape/distance
        }
        // Diagonal Moves (Capture or En Passant)
        else if dy.abs() == 1 {
            let expected_dx = if is_white { -1 } else { 1 };
            if dx != expected_dx { return false; } // Must move one step forward diagonally

            // Standard Capture
            if ending_piece != 0 && ending_piece_color != piece_color {
                return true; // Valid capture
            }
            // En Passant Capture
            if ending_piece == 0 && self.is_en_passant_move(start_x, start_y, end_x, end_y) {
                return true; // Valid en passant
            }
            return false; // Diagonal move to empty square (not en passant) or friendly piece
        }
        else {
            false
        }
    }

    fn is_valid_king_move_detailed(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, piece_color: i32) -> bool {
        let dx = end_x - start_x;
        let dy = end_y - start_y;

        
        if dx.abs() <= 1 && dy.abs() <= 1 {
            return true;
        }

        // --- Castling ---
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

                return true; // Valid Kingside Castle attempt
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
            return false; // Should not happen if called correctly
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

        // 2. Get Piece Info
        let piece = self.get_piece(start_x as usize, start_y as usize);
        if piece == 0 { return false; }
        let piece_color = utils::get_piece_color(piece);
        let piece_type = utils::get_piece_type(piece);

        // 3. Get Destination Info
        let ending_piece = self.get_piece(end_x as usize, end_y as usize);
        let ending_piece_color = utils::get_piece_color(ending_piece);

        // 4. Check Target Square (Cannot capture friendly piece)
        if ending_piece != 0 && ending_piece_color == piece_color { return false; }

        // 5. Validate Move Shape & Obstructions
        match piece_type {
            1 => { // Pawn
                let is_white = piece_color == WHITE;
                let dx = end_x - start_x;
                let dy = end_y - start_y;

                if dy == 0 { // Straight move
                    if ending_piece != 0 { return false; } // Cannot move onto piece
                    if !utils::is_valid_pawn_move(start_x, start_y, end_x, end_y, is_white) { return false; }
                     // Check obstruction for double move
                    if dx.abs() == 2 {
                         let middle_x = start_x + dx / 2;
                         if self.get_piece(middle_x as usize, start_y as usize) != 0 { return false; }
                    }
                    true
                } else if dy.abs() == 1 { // Diagonal capture
                    // NOTE: For attack checks, we consider a diagonal move valid even if the square is empty
                    // because a pawn *attacks* that square. The main is_valid_move handles the capture requirement.
                     let expected_dx = if is_white { -1 } else { 1 };
                     dx == expected_dx
                     // TODO: We DON'T check en passant *attacks* here, only direct diagonal threats.
                     // Todo: En passant possibility is handled in the main is_valid_move.
                } else {
                    false // Invalid pawn move shape
                }
            },
            2 => { // Rook
                if !utils::is_valid_rook_move(start_x, start_y, end_x, end_y) { return false; }
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            3 => { // Knight
                utils::is_valid_knight_move(start_x, start_y, end_x, end_y)
            },
            4 => { // Bishop
                if !utils::is_valid_bishop_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            5 => { // Queen
                if !utils::is_valid_queen_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            6 => { // King
                // For attack checks, we only care about the one-square move shape.
                // Castling is handled separately in the main is_valid_move.
                utils::is_valid_king_move(start_x, start_y, end_x, end_y)
            },
            _ => false
        }
    }

    #[wasm_bindgen]
    pub fn make_move(&mut self, start_x: usize, start_y: usize, end_x: usize, end_y: usize) -> Result<(), JsValue> {
        // Use i32 for internal validation logic consistency
        let (sx, sy, ex, ey) = (start_x as i32, start_y as i32, end_x as i32, end_y as i32);

        if !self.is_valid_move(sx, sy, ex, ey) {
            // Optional: Provide more specific error messages
            let piece = self.get_piece(start_x, start_y);
            let error_msg = if piece == 0 {
                "No piece at starting square.".to_string()
            } else if utils::get_piece_color(piece) != self.current_turn {
                 format!("Not {}'s turn.", if self.current_turn == WHITE { "White" } else { "Black" })
            } else if self.clone().make_move_internal(sx, sy, ex, ey, true).is_err() { // Check if it leads to check
                 "Move leaves king in check.".to_string()
            }
             else {
                format!("Invalid move shape or path for piece {} from ({},{}) to ({},{}).", piece, sx, sy, ex, ey)
            };
             return Err(JsValue::from_str(&error_msg));
        }

        // If valid, perform the move and update state
        self.make_move_internal(sx, sy, ex, ey, false)?; // Call internal helper

        Ok(())
    }

     // Internal helper to perform the move mechanics and state updates
     // `is_simulation` prevents recursive state updates during validation check
    fn make_move_internal(&mut self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, is_simulation: bool) -> Result<(), String> {
        let piece = self.board[start_x as usize][start_y as usize];
        let piece_type = utils::get_piece_type(piece);
        let piece_color = utils::get_piece_color(piece);
        // let captured_piece = self.board[end_x as usize][end_y as usize]; // Needed if tracking material/halfmove

        // --- Pre-Move State Update (before board changes affect checks) ---
        let mut next_en_passant_target: Option<(usize, usize)> = None;

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

        // 4. Handle Pawn Promotion (Default to Queen for now)
        let last_rank = if piece_color == WHITE { 0 } else { 7 };
        if piece_type == 1 && end_x == last_rank {
            let promoted_queen = if piece_color == WHITE { W_QUEEN } else { B_QUEEN };
            self.board[end_x as usize][end_y as usize] = promoted_queen;
            // TODO: In a real UI, signal here that promotion choice is needed.
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
            // If a Rook was captured on its starting square (need captured_piece info for this)
            // TODO: Add capture detection logic if needed for castling rights update

            // Set potential En Passant target for the *next* turn
             if piece_type == 1 && (end_x - start_x).abs() == 2 {
                 let target_rank = (start_x + end_x) / 2; // Rank behind the pawn
                 next_en_passant_target = Some((target_rank as usize, start_y as usize));
             }
             self.en_passant_target = next_en_passant_target; // Update for next turn


            // Switch Turn
            self.current_turn = if self.current_turn == WHITE { BLACK } else { WHITE };

            // Update clocks (TODO)
            // self.halfmove_clock = ...
            // if self.current_turn == WHITE { self.fullmove_number += 1; }
        }

        Ok(())
    }

}