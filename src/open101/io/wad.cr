module Open101
  module IO
    struct WadEntry
      getter name : String
      getter data : Bytes

      def initialize(@name : String, @data : Bytes)
      end
    end

    class WadArchive
      getter entries : Array(WadEntry)

      def initialize(@entries : Array(WadEntry))
      end

      def self.read(path : String) : WadArchive
        lines = File.read_lines(path)
        entries = lines.map do |line|
          name, payload = line.split(":", 2)
          WadEntry.new(name, payload.to_slice)
        end
        new(entries)
      end
    end
  end
end
