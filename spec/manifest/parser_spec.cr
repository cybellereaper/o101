require "../spec_helper"

describe O101::Manifest::Parser do
  it "parses and validates a manifest" do
    raw = %({"version":"1.0.0","files":[{"src":"a","dst":"b","size":2,"sha256":"#{"a" * 64}","mode":"0644"}]})
    manifest = O101::Manifest::Parser.parse!(raw)

    manifest.version.should eq("1.0.0")
    manifest.files.first.mode.should eq(420)
  end

  it "rejects invalid hashes" do
    raw = %({"version":"1.0.0","files":[{"src":"a","dst":"b","size":2,"sha256":"bad","mode":"0644"}]})
    expect_raises(O101::ValidationError) { O101::Manifest::Parser.parse!(raw) }
  end
end
