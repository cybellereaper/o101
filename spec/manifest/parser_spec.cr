require "../spec_helper"

describe Open101::Manifest::Parser do
  it "parses valid manifests" do
    raw = %({"version":"1.0","files":[{"src":"a.bin","dst":"Bin/a.bin","size":3,"sha256":"#{"a"*64}"}]})
    parsed = Open101::Manifest::Parser.parse(raw)
    parsed.version.should eq("1.0")
    parsed.files.size.should eq(1)
  end

  it "rejects empty file lists" do
    expect_raises(Open101::Manifest::ValidationError) do
      Open101::Manifest::Parser.parse(%({"version":"1.0","files":[]}))
    end
  end
end
