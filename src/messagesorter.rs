use std::{
    collections::BTreeSet,
    fs::{self, File},
    io::{BufWriter, Write},
    path::{Path, PathBuf},
};

use crate::{Error, Result};

const PROTOCOL_INFO_CLOSE_TAG: &str = "</_ProtocolInfo>";
const RECORD_OPEN_TAG: &str = "<RECORD>";
const RECORD_CLOSE_TAG: &str = "</RECORD>";

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct ProcessResult {
    pub service_id: String,
    pub service_name: String,
    pub messages: Vec<String>,
}

pub fn process(raw: &str) -> Result<ProcessResult> {
    let service_id = extract_tag(raw, r#"<ServiceID TYPE="UBYT">"#, "</ServiceID>")
        .ok_or_else(|| Error::message("messagesorter: service id tag not found"))?;
    let service_name = extract_tag(raw, r#"<ProtocolType TYPE="STR">"#, "</ProtocolType>")
        .ok_or_else(|| Error::message("messagesorter: service name tag not found"))?;

    let body = strip_protocol_info(raw)?;
    let body = strip_record_blocks(body);
    let messages = collect_messages(&body);

    Ok(ProcessResult {
        service_id: service_id.to_owned(),
        service_name: service_name.to_owned(),
        messages,
    })
}

pub fn process_file(
    input_path: &Path,
    output_dir: Option<&Path>,
) -> Result<(PathBuf, ProcessResult)> {
    let contents = fs::read_to_string(input_path)
        .map_err(|error| Error::message(format!("messagesorter: read input: {error}")))?;
    let result = process(&contents)?;

    let output_dir = output_dir
        .map(Path::to_path_buf)
        .or_else(|| input_path.parent().map(Path::to_path_buf))
        .unwrap_or_else(|| PathBuf::from("."));

    fs::create_dir_all(&output_dir)
        .map_err(|error| Error::message(format!("messagesorter: ensure output dir: {error}")))?;

    let output_path = output_dir.join(output_filename(&result.service_id, &result.service_name));
    write_numbered(&output_path, &result.messages)?;

    Ok((output_path, result))
}

pub fn output_filename(service_id: &str, service_name: &str) -> String {
    format!(
        "{}_{}.txt",
        sanitize_filename_segment(service_id),
        sanitize_filename_segment(service_name)
    )
}

pub fn write_numbered(path: &Path, messages: &[String]) -> Result<()> {
    let file = File::create(path)
        .map_err(|error| Error::message(format!("messagesorter: create output: {error}")))?;
    let mut writer = BufWriter::new(file);

    for (index, message) in messages.iter().enumerate() {
        writeln!(writer, "{}: {}", index + 1, message)
            .map_err(|error| Error::message(format!("messagesorter: write output: {error}")))?;
    }

    writer
        .flush()
        .map_err(|error| Error::message(format!("messagesorter: flush output: {error}")))
}

fn sanitize_filename_segment(value: &str) -> String {
    let mut sanitized = String::with_capacity(value.len());
    for character in value.trim().chars() {
        let replacement = match character {
            '/' | '\\' | ':' | '*' | '?' | '|' => Some('-'),
            '"' => Some('\''),
            '<' => Some('('),
            '>' => Some(')'),
            '\0' => None,
            character => Some(character),
        };
        if let Some(character) = replacement {
            sanitized.push(character);
        }
    }

    if sanitized.is_empty() {
        "unknown".to_owned()
    } else {
        sanitized
    }
}

fn strip_protocol_info(raw: &str) -> Result<&str> {
    let index = raw
        .find(PROTOCOL_INFO_CLOSE_TAG)
        .ok_or_else(|| Error::message("messagesorter: protocol info closing tag not found"))?;
    Ok(&raw[index + PROTOCOL_INFO_CLOSE_TAG.len()..])
}

fn strip_record_blocks(raw: &str) -> String {
    let mut output = String::with_capacity(raw.len());
    let mut remaining = raw;

    while let Some(start) = remaining.find(RECORD_OPEN_TAG) {
        output.push_str(&remaining[..start]);
        let record = &remaining[start + RECORD_OPEN_TAG.len()..];
        let Some(end) = record.find(RECORD_CLOSE_TAG) else {
            output.push_str(&remaining[start..]);
            return output;
        };
        remaining = &record[end + RECORD_CLOSE_TAG.len()..];
    }

    output.push_str(remaining);
    output
}

fn extract_tag<'a>(raw: &'a str, open_tag: &str, close_tag: &str) -> Option<&'a str> {
    let start = raw.find(open_tag)? + open_tag.len();
    let end = raw[start..].find(close_tag)? + start;
    Some(&raw[start..end])
}

fn collect_messages(body: &str) -> Vec<String> {
    let mut messages = BTreeSet::new();

    for line in body.lines().map(str::trim) {
        if line.is_empty()
            || line.starts_with("</")
            || !line.starts_with('<')
            || !line.ends_with('>')
        {
            continue;
        }
        if line.len() >= 2 {
            messages.insert(line[1..line.len() - 1].to_owned());
        }
    }

    messages.into_iter().collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support::TempDir;

    const SAMPLE_CAPTURE: &str = r#"<ServiceID TYPE="UBYT">123</ServiceID>
<ProtocolType TYPE="STR">ChatService</ProtocolType>
<_ProtocolInfo>
  <Metadata>stuff</Metadata>
</_ProtocolInfo>
<RECORD>
  <Ignored>This should be stripped</Ignored>
</RECORD>
<Message>Hello</Message>
<Message>World</Message>
<Message>Hello</Message>
<Message>AfterRecord</Message>
</Protocol>
"#;

    #[test]
    fn processes_capture() {
        let result = process(SAMPLE_CAPTURE).expect("process capture");
        assert_eq!(result.service_id, "123");
        assert_eq!(result.service_name, "ChatService");
        assert_eq!(
            result.messages,
            [
                "Message>AfterRecord</Message",
                "Message>Hello</Message",
                "Message>World</Message",
            ]
        );
    }

    #[test]
    fn processes_file_and_writes_numbered_output() {
        let dir = TempDir::new("messagesorter");
        let input = dir.path().join("input.xml");
        fs::write(&input, SAMPLE_CAPTURE).expect("write input");

        let (output, result) = process_file(&input, None).expect("process file");
        assert_eq!(output, dir.path().join("123_ChatService.txt"));
        assert_eq!(result.messages.len(), 3);
        assert_eq!(
            fs::read_to_string(output).expect("read output"),
            "1: Message>AfterRecord</Message\n2: Message>Hello</Message\n3: Message>World</Message\n"
        );
    }

    #[test]
    fn reports_missing_metadata() {
        assert!(process(r#"<ProtocolType TYPE="STR">Name</ProtocolType>"#).is_err());
        assert!(process(r#"<ServiceID TYPE="UBYT">id</ServiceID>"#).is_err());
        assert!(process(
            r#"<ServiceID TYPE="UBYT">id</ServiceID><ProtocolType TYPE="STR">name</ProtocolType>"#
        )
        .is_err());
    }
}
