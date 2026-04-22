require "../../spec_helper"

describe O101::Open101::Serializer::BasicBinarySerializer do
  it "round-trips ByteString and GID" do
    buffer = O101::Open101::IO::ByteBuffer.new
    serializer = O101::Open101::Serializer::BasicBinarySerializer.new(buffer)

    serializer.write_byte_string(O101::Open101::Serializer::ByteString.new("abc"))
    serializer.write_gid(O101::Open101::Serializer::GID.new(123_456_789_123_u64))

    buffer.rewind

    serializer.read_byte_string.value.should eq("abc")
    serializer.read_gid.value.should eq(123_456_789_123_u64)
  end
end
