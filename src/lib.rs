mod utils;
mod constants;

mod game;
mod validation;
mod moves;
mod fen;

use wasm_bindgen::prelude::*;

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