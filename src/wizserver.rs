use std::{
    collections::{BTreeMap, HashMap},
    io::{Read, Write},
    sync::{
        Arc, Mutex, RwLock,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
};

use flate2::{Compression, read::ZlibDecoder, write::ZlibEncoder};
use serde::{Deserialize, Serialize};
use serde_json::Value;

use crate::{Error, Result};

#[derive(Clone, Copy, Debug, Default, Deserialize, Eq, Hash, PartialEq, Serialize)]
#[serde(transparent)]
pub struct Gid(pub u64);

static NEXT_GID: AtomicU64 = AtomicU64::new(0);

pub fn next_gid() -> Gid {
    Gid(NEXT_GID.fetch_add(1, Ordering::Relaxed) + 1)
}

pub fn reset_gid() {
    NEXT_GID.store(0, Ordering::Relaxed);
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum AvatarGender {
    Neutral,
    Male,
    Female,
}

#[derive(Clone, Copy, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum AvatarRace {
    Human,
    Beast,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct CharacterCreationInfo {
    pub level: i32,
    pub school_of_focus: u32,
    pub location: String,
    pub template_id: u64,
    pub name_indices: i32,
    pub avatar_behavior: AvatarBehavior,
    pub equipment: Vec<EquippedItem>,
    pub global_id: Gid,
    pub user_id: Gid,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct AvatarBehavior {
    pub gender: AvatarGender,
    pub race: AvatarRace,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct EquippedItem {
    pub template: String,
    pub dye: String,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct LoginPlayer {
    pub characters: Vec<CharacterCreationInfo>,
    pub user_id: Gid,
}

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Inventory {
    pub backpacks: BTreeMap<String, i32>,
    pub bank: BTreeMap<String, i32>,
}

#[derive(Clone, Debug, Deserialize, PartialEq, Serialize)]
#[serde(rename_all = "camelCase")]
pub struct GameObject {
    pub global_id: Gid,
    pub character_id: Gid,
    pub template_id: Gid,
    pub location: String,
    pub metadata: BTreeMap<String, Value>,
}

pub type Message = Value;

pub trait Socket: Send + Sync {
    fn send(&self, messages: &[Message]) -> std::io::Result<()>;
}

pub struct InGameCharacter {
    pub object: GameObject,
    pub inventory: Inventory,
    pub initialised: bool,
    pub x: f32,
    pub y: f32,
    pub z: f32,
    pub direction: f32,
    pub marker_x: f32,
    pub marker_y: f32,
    pub marker_z: f32,
    pub marker_direction: f32,
    pub zone_name: String,
    pub zone_id: Gid,
    pub marker_zone_name: String,
    pub marker_zone_id: Gid,
    pub char_id: Gid,
    pub gid: Gid,
    pub mobile_id: u16,
    pub current_mana: i32,
    pub max_mana: i32,
    pub socket: Option<Arc<dyn Socket>>,
}

#[derive(Default)]
pub struct MockSocket {
    messages: Mutex<Vec<Message>>,
    fail_send: AtomicBool,
}

impl MockSocket {
    pub fn set_fail_send(&self, fail: bool) {
        self.fail_send.store(fail, Ordering::Relaxed);
    }

    pub fn drain(&self) -> Vec<Message> {
        self.messages
            .lock()
            .map(|messages| messages.clone())
            .unwrap_or_default()
    }
}

impl Socket for MockSocket {
    fn send(&self, messages: &[Message]) -> std::io::Result<()> {
        if self.fail_send.load(Ordering::Relaxed) {
            return Err(std::io::Error::other("forced send failure"));
        }
        self.messages
            .lock()
            .map_err(|_| std::io::Error::other("mock socket lock poisoned"))?
            .extend_from_slice(messages);
        Ok(())
    }
}

pub struct Zone {
    max_players: u32,
    state: RwLock<ZoneState>,
}

struct ZoneState {
    players: HashMap<Gid, Arc<Mutex<InGameCharacter>>>,
    mobile_id_counter: u16,
}

impl Zone {
    pub fn new(max_players: u32) -> Self {
        Self {
            max_players,
            state: RwLock::new(ZoneState {
                players: HashMap::new(),
                mobile_id_counter: 1,
            }),
        }
    }

    pub fn add_player(&self, player: Arc<Mutex<InGameCharacter>>) -> Result<()> {
        let mut state = self
            .state
            .write()
            .map_err(|_| Error::message("zone: lock poisoned"))?;
        if state.players.len() as u32 >= self.max_players {
            return Err(Error::message("zone capacity reached"));
        }

        let mut character = player
            .lock()
            .map_err(|_| Error::message("zone: player lock poisoned"))?;
        if character.mobile_id == 0 {
            character.mobile_id = state.mobile_id_counter;
            state.mobile_id_counter = state.mobile_id_counter.wrapping_add(1);
        }
        let gid = character.gid;
        drop(character);
        state.players.insert(gid, player);
        Ok(())
    }

    pub fn remove_player(&self, id: Gid) -> Result<Option<Arc<Mutex<InGameCharacter>>>> {
        Ok(self
            .state
            .write()
            .map_err(|_| Error::message("zone: lock poisoned"))?
            .players
            .remove(&id))
    }

    pub fn broadcast(&self, source: Gid, messages: &[Message]) -> Result<()> {
        let sockets = {
            let state = self
                .state
                .read()
                .map_err(|_| Error::message("zone: lock poisoned"))?;
            let mut sockets = Vec::new();
            for player in state.players.values() {
                let player = player
                    .lock()
                    .map_err(|_| Error::message("zone: player lock poisoned"))?;
                if player.gid != source
                    && let Some(socket) = &player.socket
                {
                    sockets.push(Arc::clone(socket));
                }
            }
            sockets
        };

        for socket in sockets {
            let _ = socket.send(messages);
        }
        Ok(())
    }

    pub fn player_count(&self) -> Result<u32> {
        Ok(self
            .state
            .read()
            .map_err(|_| Error::message("zone: lock poisoned"))?
            .players
            .len() as u32)
    }

    pub fn capacity(&self) -> u32 {
        self.max_players
    }

    pub fn for_each<F>(&self, mut callback: F) -> Result<()>
    where
        F: FnMut(&Arc<Mutex<InGameCharacter>>),
    {
        let state = self
            .state
            .read()
            .map_err(|_| Error::message("zone: lock poisoned"))?;
        for player in state.players.values() {
            callback(player);
        }
        Ok(())
    }
}

pub struct Realm {
    max_players: u32,
    zones: Vec<Arc<Zone>>,
    cache: RwLock<RealmCache>,
}

struct RealmCache {
    characters: HashMap<Gid, CharacterCreationInfo>,
    users: HashMap<Gid, Vec<Gid>>,
}

impl Realm {
    pub fn new(max_players: u32, zone_count: u32, max_players_per_zone: u32) -> Self {
        let zones = (0..zone_count)
            .map(|_| Arc::new(Zone::new(max_players_per_zone)))
            .collect();
        Self {
            max_players,
            zones,
            cache: RwLock::new(RealmCache {
                characters: HashMap::new(),
                users: HashMap::new(),
            }),
        }
    }

    pub fn max_players(&self) -> u32 {
        self.max_players
    }

    pub fn assign_zone(&self, player: Arc<Mutex<InGameCharacter>>) -> Result<()> {
        for zone in &self.zones {
            if zone.player_count()? >= zone.capacity() {
                continue;
            }
            if zone.add_player(Arc::clone(&player)).is_ok() {
                return Ok(());
            }
        }
        Err(Error::message("no zone capacity available"))
    }

    pub fn broadcast(&self, source: Gid, message: Message) -> Result<()> {
        for zone in &self.zones {
            zone.broadcast(source, std::slice::from_ref(&message))?;
        }
        Ok(())
    }

    pub fn cache_character_creation_info(&self, info: CharacterCreationInfo) -> Result<()> {
        let mut cache = self
            .cache
            .write()
            .map_err(|_| Error::message("realm: cache lock poisoned"))?;
        cache.characters.insert(info.global_id, info.clone());
        cache
            .users
            .entry(info.user_id)
            .or_default()
            .push(info.global_id);
        Ok(())
    }

    pub fn character_from_cache(&self, id: Gid) -> Result<Option<CharacterCreationInfo>> {
        Ok(self
            .cache
            .read()
            .map_err(|_| Error::message("realm: cache lock poisoned"))?
            .characters
            .get(&id)
            .cloned())
    }

    pub fn characters_for_player(&self, player_id: Gid) -> Result<Vec<CharacterCreationInfo>> {
        let ids = self
            .cache
            .read()
            .map_err(|_| Error::message("realm: cache lock poisoned"))?
            .users
            .get(&player_id)
            .cloned()
            .unwrap_or_default();

        if ids.is_empty() {
            let mut info = default_character_info();
            info.user_id = player_id;
            info.global_id = next_gid();
            return Ok(vec![info]);
        }

        let mut characters = Vec::with_capacity(ids.len());
        for id in ids {
            if let Some(mut info) = self.character_from_cache(id)? {
                info.user_id = player_id;
                characters.push(info);
            }
        }
        Ok(characters)
    }

    pub fn zones(&self) -> Vec<Arc<Zone>> {
        self.zones.clone()
    }
}

pub fn bytes_equal(left: &[u8], right: &[u8]) -> bool {
    if left.len() != right.len() {
        return false;
    }
    let mut difference = 0_u8;
    for (left, right) in left.iter().zip(right) {
        difference |= left ^ right;
    }
    difference == 0
}

pub fn parse_chat_message(packet: &[u8]) -> Result<String> {
    if packet.len() < 4 {
        return Err(Error::message("packet too short"));
    }

    let char_count = u16::from_le_bytes([packet[2], packet[3]]) as usize;
    let bytes_needed = char_count
        .checked_mul(2)
        .ok_or_else(|| Error::message("declared length exceeds packet size"))?;
    if packet.len().saturating_sub(4) < bytes_needed {
        return Err(Error::message("declared length exceeds packet size"));
    }

    let mut chars = Vec::with_capacity(char_count);
    for chunk in packet[4..4 + bytes_needed].chunks_exact(2) {
        chars.push(u16::from_le_bytes([chunk[0], chunk[1]]));
    }
    while chars.last() == Some(&0) {
        chars.pop();
    }

    Ok(char::decode_utf16(chars)
        .map(|value| value.unwrap_or(char::REPLACEMENT_CHARACTER))
        .collect())
}

pub fn hex_string_to_bytes(value: &str) -> Result<Vec<u8>> {
    let value = value.replace(' ', "");
    if value.len() % 2 != 0 {
        return Err(Error::message(
            "hex string must contain an even amount of characters",
        ));
    }

    value
        .as_bytes()
        .chunks_exact(2)
        .map(|pair| {
            let pair = std::str::from_utf8(pair)
                .map_err(|error| Error::message(format!("invalid hex string: {error}")))?;
            u8::from_str_radix(pair, 16)
                .map_err(|error| Error::message(format!("invalid hex string: {error}")))
        })
        .collect()
}

pub fn compress(data: &[u8]) -> Result<Vec<u8>> {
    let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(data)?;
    Ok(encoder.finish()?)
}

pub fn decompress(data: &[u8]) -> Result<Vec<u8>> {
    let mut decoder = ZlibDecoder::new(data);
    let mut output = Vec::new();
    decoder.read_to_end(&mut output)?;
    Ok(output)
}

pub fn serialize_player(object: &GameObject, new_object: bool) -> Result<Vec<u8>> {
    serialize_json(object, new_object)
}

pub fn serialize_item(object: &BTreeMap<String, Value>, new_object: bool) -> Result<Vec<u8>> {
    serialize_json(object, new_object)
}

fn serialize_json<T: Serialize>(object: &T, new_object: bool) -> Result<Vec<u8>> {
    let payload = serde_json::to_vec(object)?;
    if new_object {
        return Ok(payload);
    }

    let compressed = compress(&payload)?;
    let mut framed = Vec::with_capacity(4 + compressed.len());
    framed.extend_from_slice(&(payload.len() as u32).to_le_bytes());
    framed.extend_from_slice(&compressed);
    Ok(framed)
}

pub fn default_character_info() -> CharacterCreationInfo {
    CharacterCreationInfo {
        level: 1,
        school_of_focus: 0x04f836b3,
        location: "WizardCity/WC_Ravenwood".to_owned(),
        template_id: 1,
        name_indices: 0,
        avatar_behavior: AvatarBehavior {
            gender: AvatarGender::Neutral,
            race: AvatarRace::Human,
        },
        equipment: Vec::new(),
        global_id: next_gid(),
        user_id: Gid::default(),
    }
}

pub fn character_creation_to_game_object(info: &CharacterCreationInfo) -> GameObject {
    let mut metadata = BTreeMap::from([
        ("level".to_owned(), Value::from(info.level)),
        (
            "schoolOfFocus".to_owned(),
            Value::from(info.school_of_focus),
        ),
        ("nameIndices".to_owned(), Value::from(info.name_indices)),
        (
            "gender".to_owned(),
            serde_json::to_value(info.avatar_behavior.gender)
                .expect("enum serialization cannot fail"),
        ),
        (
            "race".to_owned(),
            serde_json::to_value(info.avatar_behavior.race)
                .expect("enum serialization cannot fail"),
        ),
    ]);
    if !info.equipment.is_empty() {
        metadata.insert(
            "equipment".to_owned(),
            serde_json::to_value(&info.equipment).expect("equipment serialization cannot fail"),
        );
    }

    GameObject {
        global_id: info.global_id,
        character_id: info.global_id,
        template_id: Gid(info.template_id),
        location: info.location.clone(),
        metadata,
    }
}

pub fn create_player_instance(info: &CharacterCreationInfo) -> InGameCharacter {
    InGameCharacter {
        object: character_creation_to_game_object(info),
        inventory: Inventory {
            backpacks: BTreeMap::from([("cards".to_owned(), 0)]),
            bank: BTreeMap::new(),
        },
        initialised: true,
        x: 1132.0,
        y: 3.0,
        z: 3.0,
        direction: 0.0,
        marker_x: 0.0,
        marker_y: 0.0,
        marker_z: 0.0,
        marker_direction: 0.0,
        zone_name: "WizardCity/WC_Ravenwood".to_owned(),
        zone_id: Gid(123004564835992122),
        marker_zone_name: "WizardCity/WC_Ravenwood".to_owned(),
        marker_zone_id: Gid(123004564835992122),
        char_id: info.global_id,
        gid: info.global_id,
        mobile_id: 0,
        current_mana: 15,
        max_mana: 15,
        socket: None,
    }
}

pub fn create_default_player_instance() -> InGameCharacter {
    create_player_instance(&default_character_info())
}

#[cfg(test)]
mod tests {
    use serde_json::json;

    use super::*;

    #[test]
    fn zone_assigns_mobile_ids_and_enforces_capacity() {
        reset_gid();
        let zone = Zone::new(2);
        let first = Arc::new(Mutex::new(create_default_player_instance()));
        let second = Arc::new(Mutex::new(create_default_player_instance()));
        zone.add_player(Arc::clone(&first)).unwrap();
        zone.add_player(Arc::clone(&second)).unwrap();

        let first_id = first.lock().unwrap().mobile_id;
        let second_id = second.lock().unwrap().mobile_id;
        assert_ne!(first_id, 0);
        assert_ne!(first_id, second_id);

        let third = Arc::new(Mutex::new(create_default_player_instance()));
        assert!(zone.add_player(third).is_err());
    }

    #[test]
    fn zone_broadcast_skips_source() {
        reset_gid();
        let zone = Zone::new(3);
        let mut players = Vec::new();
        let mut sockets = Vec::new();

        for _ in 0..3 {
            let socket = Arc::new(MockSocket::default());
            let mut player = create_default_player_instance();
            player.socket = Some(socket.clone());
            let player = Arc::new(Mutex::new(player));
            zone.add_player(Arc::clone(&player)).unwrap();
            players.push(player);
            sockets.push(socket);
        }

        let source = players[0].lock().unwrap().gid;
        let message = json!({"text":"hello"});
        zone.broadcast(source, std::slice::from_ref(&message))
            .unwrap();
        assert!(sockets[0].drain().is_empty());
        assert_eq!(sockets[1].drain(), vec![message.clone()]);
        assert_eq!(sockets[2].drain(), vec![message]);
    }

    #[test]
    fn realm_distributes_players_and_caches_characters() {
        reset_gid();
        let realm = Realm::new(4, 2, 1);
        for _ in 0..2 {
            realm
                .assign_zone(Arc::new(Mutex::new(create_default_player_instance())))
                .unwrap();
        }
        assert_eq!(realm.zones()[0].player_count().unwrap(), 1);
        assert_eq!(realm.zones()[1].player_count().unwrap(), 1);
        assert!(
            realm
                .assign_zone(Arc::new(Mutex::new(create_default_player_instance())))
                .is_err()
        );

        let mut info = default_character_info();
        info.user_id = next_gid();
        realm.cache_character_creation_info(info.clone()).unwrap();
        assert_eq!(
            realm.character_from_cache(info.global_id).unwrap(),
            Some(info.clone())
        );
        assert_eq!(
            realm.characters_for_player(info.user_id).unwrap(),
            vec![info]
        );
    }

    #[test]
    fn utility_round_trips() {
        assert!(bytes_equal(&[1, 2, 3], &[1, 2, 3]));
        assert!(!bytes_equal(&[1, 2, 3], &[1, 2, 4]));
        assert_eq!(
            hex_string_to_bytes("DE AD BE EF").unwrap(),
            [0xde, 0xad, 0xbe, 0xef]
        );

        let data = b"wizard101";
        assert_eq!(decompress(&compress(data).unwrap()).unwrap(), data);

        let chars = [b'h' as u16, b'i' as u16, 0];
        let mut packet = vec![0x10, 0x20];
        packet.extend_from_slice(&(chars.len() as u16).to_le_bytes());
        for character in chars {
            packet.extend_from_slice(&character.to_le_bytes());
        }
        assert_eq!(parse_chat_message(&packet).unwrap(), "hi");
    }

    #[test]
    fn serialization_framing_matches_payload_size() {
        reset_gid();
        let info = default_character_info();
        let object = character_creation_to_game_object(&info);
        let raw = serialize_player(&object, true).unwrap();
        assert!(!raw.is_empty());

        let framed = serialize_player(&object, false).unwrap();
        let declared = u32::from_le_bytes(framed[..4].try_into().unwrap()) as usize;
        let decoded = decompress(&framed[4..]).unwrap();
        assert_eq!(decoded.len(), declared);

        let item = BTreeMap::from([
            ("id".to_owned(), json!(42)),
            ("name".to_owned(), json!("amulet")),
        ]);
        assert!(!serialize_item(&item, false).unwrap().is_empty());
    }
}
