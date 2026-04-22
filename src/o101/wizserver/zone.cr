module O101
  module WizServer
    class Zone
      getter id : Int32
      getter capacity : Int32

      def initialize(@id : Int32, @capacity : Int32)
        @players = Set(String).new
      end

      def add(character_id : String) : Bool
        return false if full?
        return false if @players.includes?(character_id)
        @players.add(character_id)
        true
      end

      def remove(character_id : String)
        @players.delete(character_id)
      end

      def size : Int32
        @players.size
      end

      def full? : Bool
        @players.size >= capacity
      end
    end
  end
end
