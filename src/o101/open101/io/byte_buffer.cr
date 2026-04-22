module O101
  module Open101
    module IO
      class ByteBuffer
        getter position : Int32

        def initialize(capacity = 128)
          @bytes = ::Bytes.new(capacity)
          @length = 0
          @position = 0
        end

        def write_u8(value : UInt8)
          ensure_capacity(1)
          @bytes[@length] = value
          @length += 1
        end

        def write_i32(value : Int32)
          4.times { |i| write_u8(((value >> (8 * i)) & 0xff).to_u8) }
        end

        def write_string(value : String)
          payload = value.to_slice
          write_i32(payload.size)
          payload.each { |b| write_u8(b) }
        end

        def rewind
          @position = 0
        end

        def read_u8 : UInt8
          raise Error.new("buffer underflow") if @position >= @length
          value = @bytes[@position]
          @position += 1
          value
        end

        def read_i32 : Int32
          b0 = read_u8.to_i
          b1 = read_u8.to_i << 8
          b2 = read_u8.to_i << 16
          b3 = read_u8.to_i << 24
          b0 | b1 | b2 | b3
        end

        def read_string : String
          size = read_i32
          raise Error.new("invalid string size") if size < 0
          data = Bytes.new(size)
          size.times { |i| data[i] = read_u8 }
          String.new(data)
        end

        def to_slice : Bytes
          @bytes[0, @length]
        end

        private def ensure_capacity(increment : Int32)
          needed = @length + increment
          return if needed <= @bytes.size

          new_size = @bytes.size
          while new_size < needed
            new_size *= 2
          end
          grown = ::Bytes.new(new_size)
          grown.copy_from(@bytes)
          @bytes = grown
        end
      end
    end
  end
end
