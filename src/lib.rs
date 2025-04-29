mod utils;
use wasm_bindgen::prelude::*;

#[wasm_bindgen]
pub struct ChessGame {
    board: Vec<Vec<i32>>,
    // TODO: Add state like current turn, castling rights, en passant target, etc.
}

#[wasm_bindgen]
impl ChessGame {
    #[wasm_bindgen(constructor)] // Use constructor for JS `new ChessGame()`
    pub fn new() -> ChessGame {
        let mut board = vec![vec![0; 8]; 8];

        // Standard piece numbering:
        // White: 1=Pawn, 2=Rook, 3=Knight, 4=Bishop, 5=Queen, 6=King
        // Black: 7=Pawn, 8=Rook, 9=Knight, 10=Bishop, 11=Queen, 12=King

        // Black pieces (Rank 8 -> board index 0)
        board[0] = vec![8, 9, 10, 11, 12, 10, 9, 8]; // B_Rook, B_Knight, B_Bishop, B_Queen, B_King, B_Bishop, B_Knight, B_Rook
        // Black Pawns (Rank 7 -> board index 1)
        board[1] = vec![7, 7, 7, 7, 7, 7, 7, 7];     // B_Pawn * 8
        // White Pawns (Rank 2 -> board index 6)
        board[6] = vec![1, 1, 1, 1, 1, 1, 1, 1];     // W_Pawn * 8
        // White pieces (Rank 1 -> board index 7)
        board[7] = vec![2, 3, 4, 5, 6, 4, 3, 2];     // W_Rook, W_Knight, W_Bishop, W_Queen, W_King, W_Bishop, W_Knight, W_Rook

        ChessGame { board }
    }

    // Return a serialized JSON string
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

    /// Attempts to move a piece. Returns true if the move was made, false otherwise.
    /// Does NOT currently validate the move based on game rules, only placement.
    /// For rule validation, call is_valid_move first.
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

        // Basic move - just updates the board.
        // TODO: Handle captures (e.g., removing captured piece) if needed outside validation.
        // TODO: Handle special moves like castling, en passant, promotion.
        self.board[end_x][end_y] = piece;
        self.board[start_x][start_y] = 0;
        true
    }


    /// Checks if a move is valid according to chess rules.
    /// Includes basic shape validation, capture rules, and obstruction checks.
    /// NOTE: Does NOT check for putting the own king in check.
    /// NOTE: Does NOT handle special moves (castling, en passant, promotion).
    pub fn is_valid_move(&self, start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
        // 1. Check Bounds
        if !Self::is_on_board(start_x, start_y) || !Self::is_on_board(end_x, end_y) {
            return false;
        }
        // Cannot move to the same square
        if start_x == end_x && start_y == end_y {
            return false;
        }

        // 2. Get Piece Info
        let piece = self.get_piece(start_x as usize, start_y as usize);
        if piece == 0 {
            return false; // No piece to move
        }
        let piece_color = utils::get_piece_color(piece);
        let piece_type = utils::get_piece_type(piece);

        // 3. Get Destination Info
        let ending_piece = self.get_piece(end_x as usize, end_y as usize);
        let ending_piece_color = utils::get_piece_color(ending_piece);

        // 4. Check Target Square (Capture Rule)
        if ending_piece != 0 && ending_piece_color == piece_color {
            return false; // Cannot capture a friendly piece
        }
        // Now we know the target is either empty or occupied by an opponent

        // 5. Validate Move Shape & Obstructions
        match piece_type {
            // --- Pawn ---
            1 => { // Pawn
                let is_white = piece_color == 1;
                let dx = end_x - start_x; // Change in row index
                let dy = end_y - start_y; // Change in column index

                // Check Standard Forward Moves (must land on empty square)
                if dy == 0 { // Moving straight
                    if ending_piece != 0 { return false; } // Cannot move forward onto a piece
                    if !utils::is_valid_pawn_move(start_x, start_y, end_x, end_y, is_white) {
                         return false; // Basic forward move shape invalid
                    }
                    // Check obstruction for double move
                    if dx.abs() == 2 {
                         let middle_x = start_x + dx / 2;
                         if self.get_piece(middle_x as usize, start_y as usize) != 0 {
                             return false; // Path blocked
                         }
                    }
                     true // Valid forward move
                }
                // Check Captures (must land on opponent piece)
                else if dy.abs() == 1 { // Moving diagonal
                    if ending_piece == 0 || ending_piece_color == piece_color { return false; } // Must capture opponent

                    // Check if shape matches capture direction
                     let expected_dx = if is_white { -1 } else { 1 };
                     dx == expected_dx // Correct forward direction and diagonal
                }
                // Check En Passant (TODO)
                else {
                    false // Invalid pawn move (e.g., sideways)
                }
            },
            // --- Rook ---
            2 => { // Rook
                if !utils::is_valid_rook_move(start_x, start_y, end_x, end_y) { return false; }
                self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            // --- Knight ---
            3 => { // Knight
                // Knights jump, no obstruction check needed
                utils::is_valid_knight_move(start_x, start_y, end_x, end_y)
            },
            // --- Bishop ---
            4 => { // Bishop
                if !utils::is_valid_bishop_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            // --- Queen ---
            5 => { // Queen
                if !utils::is_valid_queen_move(start_x, start_y, end_x, end_y) { return false; }
                 self.is_path_clear(start_x, start_y, end_x, end_y)
            },
            // --- King ---
            6 => { // King
                // Basic move validation (no castling check here)
                 // Check if the move puts the king in check (TODO: Complex check needed)
                utils::is_valid_king_move(start_x, start_y, end_x, end_y)
            },
            _ => false // Should not happen with valid piece codes
        }

        // 6. Check if Move Puts Own King in Check (TODO - Important but complex)
        // This requires simulating the move and then calling is_check for the current player.
        // For now, we assume the move is valid if it passes the above checks.
        // return true; // (Remove this line when implementing the self-check)
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
        let king_piece = if color == 1 { 6 } else { 12 }; // White King = 6, Black King = 12
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
            if king_x != -1 { // Found the king
                break;
            }
        }

        if king_x == -1 {
            // Should not happen in a valid game state, maybe return error or panic?
             utils::log(&format!("King for color {} not found on board!", color));
            return false; // Or true? Depends on how you want to handle invalid state.
        }

        // 2. Check if any opponent piece can attack the king's square
        let opponent_color = if color == 1 { 2 } else { 1 };
        for r in 0..8 {
            for c in 0..8 {
                let piece = self.board[r][c];
                if piece != 0 && utils::get_piece_color(piece) == opponent_color {
                    // Found an opponent piece. Check if it can move to the king's square.
                    // Use the updated `is_valid_move` which now handles captures and obstructions.
                    if self.is_valid_move(r as i32, c as i32, king_x, king_y) {
                        // An opponent piece has a valid move to the king's square.
                        return true; // King is in check
                    }
                }
            }
        }

        // No opponent piece found that can attack the king.
        false // King is not in check
    }

    // TODO: Add functions for:
    // - Checkmate detection
    // - Stalemate detection
    // - Handling turns
    // - Castling logic
    // - En passant logic
    // - Pawn promotion
    // - Game state (whose turn, game over, etc.)
}

// Optional: Add a simple logging function in JS (e.g., in utils.js or directly)
// And declare it here if you want to use it from Rust for debugging:
// #[wasm_bindgen(module = "/utils.js")] // Adjust path if needed
// extern "C" {
//     #[wasm_bindgen(js_namespace = console)]
//     fn log(s: &str);
// }

// Or using web_sys::console
// fn log(s: &str) {
//     web_sys::console::log_1(&s.into());
// }