module O101
  module WizServer
    class Realm
      getter max_players : Int32

      def initialize(zone_count : Int32, zone_capacity : Int32, @max_players : Int32)
        @zones = Array(Zone).new(zone_count) { |i| Zone.new(i.to_i, zone_capacity) }
        @characters = Hash(String, InGameCharacter).new
      end

      def upsert_character(character : InGameCharacter)
        @characters[character.id] = character
      end

      def assign_zone(character_id : String) : Zone
        raise Error.new("realm is full") if online_count >= max_players

        zone = @zones.min_by(&.size)
        raise Error.new("all zones are full") unless zone.add(character_id)

        zone
      end

      def online_count : Int32
        @zones.sum(&.size)
      end

      def character?(id : String) : InGameCharacter?
        @characters[id]?
      end
    end
  end
end
