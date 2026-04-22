module Open101
  module WizServer
    class Zone
      getter id : UInt32
      getter capacity : UInt32
      getter players = Hash(UInt32, Character).new

      def initialize(@id : UInt32, @capacity : UInt32)
      end

      def full? : Bool
        @players.size >= @capacity
      end

      def add(character : Character) : Bool
        return false if full?
        @players[character.id] = character
        true
      end

      def remove(character_id : UInt32)
        @players.delete(character_id)
      end
    end
  end
end
