module Open101
  module WizServer
    class Realm
      getter max_players : UInt32
      getter zones : Array(Zone)

      def initialize(@max_players : UInt32, zone_count : UInt32, zone_capacity : UInt32)
        @zones = Array(Zone).new
        zone_count.times { |i| @zones << Zone.new(i.to_u32, zone_capacity) }
      end

      def assign(character : Character) : Zone?
        @zones.find { |zone| zone.add(character) }
      end
    end
  end
end
