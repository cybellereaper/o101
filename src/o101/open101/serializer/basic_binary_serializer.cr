module O101
  module Open101
    module Serializer
      class BasicBinarySerializer
        def initialize(@buffer : Open101::IO::ByteBuffer)
        end

        def write_byte_string(value : ByteString)
          @buffer.write_string(value.value)
        end

        def read_byte_string : ByteString
          ByteString.new(@buffer.read_string)
        end

        def write_gid(gid : GID)
          low = (gid.value & 0xffffffff).to_u32
          high = ((gid.value >> 32) & 0xffffffff).to_u32
          @buffer.write_i32(low.unsafe_as(Int32))
          @buffer.write_i32(high.unsafe_as(Int32))
        end

        def read_gid : GID
          low = @buffer.read_i32.unsafe_as(UInt32)
          high = @buffer.read_i32.unsafe_as(UInt32)
          GID.new(((high.to_u64 << 32) | low.to_u64))
        end
      end
    end
  end
end
