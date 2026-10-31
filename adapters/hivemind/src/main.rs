//! Reads every corpus case with Hivemind OS's SKILL.md parser (vendored verbatim
//! from crates/hive-skills/src/parser.rs) and prints one JSON reading per line.
#[path = "../vendor/parser.rs"]
#[allow(dead_code)]
mod parser;

use serde_json::json;
use std::fs;
use std::path::Path;

fn main() {
    let root = std::env::args().nth(1).expect("usage: hivemind-read <corpus-dir>");
    let mut cases: Vec<_> = fs::read_dir(&root).unwrap().flatten().filter(|e| e.path().is_dir()).collect();
    cases.sort_by_key(|e| e.file_name());
    for c in cases {
        let case = c.file_name().to_string_lossy().to_string();
        let reading = match fs::read_to_string(skill_file(&c.path())) {
            Err(e) => json!({"loader": "hivemind", "case": case, "loaded": false, "error": e.to_string()}),
            Ok(raw) => match parser::parse_skill_md(&raw) {
                Ok(s) => json!({"loader": "hivemind", "case": case, "loaded": true,
                    "name": s.manifest.name, "description": s.manifest.description, "body": s.body}),
                Err(e) => json!({"loader": "hivemind", "case": case, "loaded": false, "error": e.to_string()}),
            },
        };
        println!("{reading}");
    }
}

fn skill_file(case_dir: &Path) -> std::path::PathBuf {
    let mut dirs: Vec<_> = fs::read_dir(case_dir).unwrap().flatten().filter(|e| e.path().is_dir()).collect();
    dirs.sort_by_key(|e| e.file_name());
    dirs[0].path().join("SKILL.md")
}
