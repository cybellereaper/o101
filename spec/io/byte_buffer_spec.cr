require "../spec_helper"

describe Open101::IO::ByteBuffer do
  it "writes and reads values in little-endian order" do
    buf = Open101::IO::ByteBuffer.new
    buf.write_u8(5_u8)
    buf.write_i32(1024)
    buf.read_u8.should eq(5_u8)
    buf.read_i32.should eq(1024)
  end
end
