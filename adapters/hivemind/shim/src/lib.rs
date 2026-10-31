//! Only the SkillManifest fields that hive-skills' parser.rs constructs, copied from
//! hive-contracts/src/skills.rs at hivemind-os/hivemind@4b99f02 (serde attributes dropped).
use std::collections::BTreeMap;

#[derive(Debug, Clone)]
pub struct SkillManifest {
    pub name: String,
    pub description: String,
    pub license: Option<String>,
    pub compatibility: Option<String>,
    pub metadata: BTreeMap<String, String>,
    pub allowed_tools: Option<String>,
}
