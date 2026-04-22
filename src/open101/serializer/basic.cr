module Open101
  module Serializer
    module Basic
      def self.encode_string(value : String) : Bytes
        size = value.bytesize
        buffer = Bytes.new(size + 2)
        buffer[0] = (size & 0xFF).to_u8
        buffer[1] = ((size >> 8) & 0xFF).to_u8
        value.to_slice.each_with_index { |byte, i| buffer[i + 2] = byte }
        buffer
      end

      def self.decode_string(data : Bytes) : String
        raise "invalid payload" if data.size < 2
        size = data[0].to_i + (data[1].to_i << 8)
        raise "invalid payload size" if data.size < size + 2
        String.new(data[2, size])
      end
    end
  end
end
