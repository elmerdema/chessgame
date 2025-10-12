mod utils;
mod constants;

use wasm_bindgen::prelude::*;
use wasm_bindgen::JsValue;
use constants::{
    W_PAWN, W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING,
    B_PAWN, B_ROOK, B_KNIGHT, B_BISHOP, B_QUEEN, B_KING,
    WHITE, BLACK
};

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
        board[0] = vec![B_ROOK,B_KNIGHT, B_BISHOP, B_QUEEN, B_KING, B_BISHOP, B_KNIGHT, B_ROOK];
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

        // Call the internal helper to simulate the move. It won't update state or turn.
        if let Err(_) = temp_game.make_move_internal(start_x, start_y, end_x, end_y, true) {
            // This should ideally not happen if is_basic_move_valid is true, but for safety
            return false; 
        }

        // Now, check if the king of the player who moved is in check in the temp state
        if temp_game.is_check(piece_color) {
            utils::log("Move failed: leaves king in check");
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

            // Standard Capture
            if ending_piece != 0 && ending_piece_color != piece_color {
                utils::log("Valid pawn capture");
                return true;
            }
            // En Passant Capture
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
    pub fn make_move(&mut self, start_x: usize, start_y: usize, end_x: usize, end_y: usize) -> Result<Option<Vec<usize>>, JsValue> {

        let (sx, sy, ex, ey) = (start_x as i32, start_y as i32, end_x as i32, end_y as i32);

        // Get piece info *before* the move, for validation and then for promotion check
        let piece_at_start = self.get_piece(start_x, start_y);
        let piece_type_at_start = utils::get_piece_type(piece_at_start);
        let piece_color_at_start = utils::get_piece_color(piece_at_start);

        if !self.is_valid_move(sx, sy, ex, ey) {
            // Error handling remains similar, but the "leaves king in check" is now handled
            // by `is_valid_move` returning false directly.
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
        // Check for promotion *after* the move has been applied to the board.
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
    fn make_move_internal(&mut self, start_x: i32, start_y: i32, end_x: i32, end_y: i32, is_simulation: bool) -> Result<(), String> {
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


    #[wasm_bindgen]
    pub fn fen(&self) -> String {
        let mut fen_string = String::new();

        // piece Piece Placement
        for r in 0..8 {
            let mut empty_squares = 0;
            for c in 0..8 {
                let piece = self.board[r][c];
                if piece == 0 {
                    empty_squares += 1;
                } else {
                    if empty_squares > 0 {
                        fen_string.push_str(&empty_squares.to_string());
                        empty_squares = 0;
                    }
                    let piece_char = match piece {
                        W_PAWN => 'P', W_ROOK => 'R', W_KNIGHT => 'N', W_BISHOP => 'B', W_QUEEN => 'Q', W_KING => 'K',
                        B_PAWN => 'p', B_ROOK => 'r', B_KNIGHT => 'n', B_BISHOP => 'b', B_QUEEN => 'q', B_KING => 'k',
                        _ => '?',
                    };
                    fen_string.push(piece_char);
                }
            }
            if empty_squares > 0 {
                fen_string.push_str(&empty_squares.to_string());
            }
            if r < 7 {
                fen_string.push('/');
            }
        }

        //  Active Color
        fen_string.push(' ');
        fen_string.push(if self.current_turn == WHITE { 'w' } else { 'b' });

        //  Castling Availability
        fen_string.push(' ');
        let mut castling_availability = String::new();
        if self.white_can_castle_kingside { castling_availability.push('K'); }
        if self.white_can_castle_queenside { castling_availability.push('Q'); }
        if self.black_can_castle_kingside { castling_availability.push('k'); }
        if self.black_can_castle_queenside { castling_availability.push('q'); }
        if castling_availability.is_empty() {
            fen_string.push('-');
        } else {
            fen_string.push_str(&castling_availability);
        }

        // En Passant Target
        fen_string.push(' ');
        if let Some((r, c)) = self.en_passant_target {
            let file = (c as u8 + b'a') as char;
            let rank = 8 - r;
            fen_string.push(file);
            fen_string.push_str(&rank.to_string());
        } else {
            fen_string.push('-');
        }

        // Halfmove Clock & 6. Fullmove Number (placeholders)
        fen_string.push_str(" 0 1");

        fen_string
    }
    #[wasm_bindgen]
pub fn load_fen(&mut self, fen:&str) -> Result<(), JsValue> {
    let parts: Vec<&str> = fen.split_whitespace().collect();
    if parts.len() < 4 {
        return Err(JsValue::from_str("Invalid FEN string"));

    }

    let mut new_board = vec![vec![0; 8]; 8];
    let piece_placement = parts[0];
    let mut row = 0;
    for rank in piece_placement.split('/') {
        let mut col = 0;
        for ch in rank.chars() {
            if col >= 8 { break; }
            if let Some(digit) = ch.to_digit(10) {
                col += digit as usize;
            } else {
                let piece = match ch {
                    'p' => B_PAWN, 'r' => B_ROOK, 'n' => B_KNIGHT, 'b' => B_BISHOP, 'q' => B_QUEEN, 'k' => B_KING,
                    'P' => W_PAWN, 'R' => W_ROOK, 'N' => W_KNIGHT, 'B' => W_BISHOP, 'Q' => W_QUEEN, 'K' => W_KING,
                    _ => 0,
                };
                if piece != 0 {
                    new_board[row][col] = piece;
                }
                col += 1;
            }
        }
        row += 1;
        if row >= 8 { break; }
    }
    self.board = new_board;

    self.current_turn = if parts[1]== "w" { WHITE} else {BLACK}; //active color 

    //castling
    let castling = parts[2];
    self.white_can_castle_kingside = castling.contains('K');
    self.white_can_castle_queenside = castling.contains('Q');
    self.black_can_castle_kingside = castling.contains('k');
    self.black_can_castle_queenside = castling.contains('q');

    let en_passant = parts[3];
    if en_passant == "-" {
        self.en_passant_target = None;
    } else {
        let file = en_passant.chars().nth(0).unwrap() as usize - 'a' as usize;
        let rank = 8 - en_passant.chars().nth(1).unwrap().to_digit(10).unwrap() as usize;
        self.en_passant_target = Some((rank, file));
    }
    // TODO:
    // Ignoring halfmove clock (parts[4]) and fullmove number (parts[5]) for now

    Ok(())


}
}