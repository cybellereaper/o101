module Open101
  module Serializer
    struct ByteString
      getter value : String

      def initialize(@value : String)
      end

      def to_bytes : Bytes
        Basic.encode_string(@value)
      end
    end

    struct Gid
      getter high : UInt32
      getter low : UInt32

      def initialize(@high : UInt32, @low : UInt32)
      end

      def to_s(io : ::IO)
        io << "#{@high}:#{@low}"
      end
    end
  end
end
