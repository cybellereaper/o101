require "../spec_helper"

describe Open101::Serializer::Basic do
  it "encodes and decodes strings" do
    encoded = Open101::Serializer::Basic.encode_string("wiz")
    Open101::Serializer::Basic.decode_string(encoded).should eq("wiz")
  end
end
