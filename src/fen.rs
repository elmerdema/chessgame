use wasm_bindgen::prelude::*;
use wasm_bindgen::JsValue;

use crate::constants::{
    W_PAWN, W_ROOK, W_KNIGHT, W_BISHOP, W_QUEEN, W_KING,
    B_PAWN, B_ROOK, B_KNIGHT, B_BISHOP, B_QUEEN, B_KING,
    WHITE, BLACK
};
use crate::ChessGame;

#[wasm_bindgen]
impl ChessGame {
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

    self.current_turn = if parts[1]== "w" { WHITE} else {BLACK};

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
