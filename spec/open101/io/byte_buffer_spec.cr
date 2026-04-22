require "../../spec_helper"

describe O101::Open101::IO::ByteBuffer do
  it "round-trips primitives" do
    buffer = O101::Open101::IO::ByteBuffer.new
    buffer.write_u8(10_u8)
    buffer.write_i32(1337)
    buffer.write_string("wizard")
    buffer.rewind

    buffer.read_u8.should eq(10_u8)
    buffer.read_i32.should eq(1337)
    buffer.read_string.should eq("wizard")
  end

  it "raises on underflow" do
    expect_raises(O101::Error) { O101::Open101::IO::ByteBuffer.new.read_u8 }
  end
end
