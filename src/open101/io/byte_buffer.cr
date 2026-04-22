module Open101
  module IO
    class ByteBuffer
      @buffer : Array(UInt8)
      @position : Int32

      def initialize(initial = Bytes.empty)
        @buffer = initial.to_a
        @position = 0
      end

      def write_u8(value : UInt8)
        @buffer << value
      end

      def read_u8 : UInt8
        raise "buffer underflow" if @position >= @buffer.size
        value = @buffer[@position]
        @position += 1
        value
      end

      def write_i32(value : Int32)
        4.times { |i| write_u8(((value >> (8 * i)) & 0xFF).to_u8) }
      end

      def read_i32 : Int32
        result = 0
        4.times { |i| result |= (read_u8.to_i32 << (8 * i)) }
        result
      end

      def to_slice : Bytes
        Bytes.new(@buffer.size) { |i| @buffer[i] }
      end
    end
  end
end
