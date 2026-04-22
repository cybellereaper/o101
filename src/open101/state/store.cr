module Open101
  module State
    struct FileInfo
      include JSON::Serializable
      getter size : Int64
      getter sha256 : String

      def initialize(@size : Int64, @sha256 : String)
      end
    end

    struct Snapshot
      include JSON::Serializable
      getter version : String?
      getter files : Hash(String, FileInfo)

      def initialize(@version : String? = nil, @files : Hash(String, FileInfo) = Hash(String, FileInfo).new)
      end
    end

    class Store
      getter path : String
      @lock = Mutex.new

      def initialize(@path : String)
        raise ArgumentError.new("state path is required") if @path.strip.empty?
      end

      def load : Snapshot
        @lock.synchronize do
          return Snapshot.new if !File.exists?(@path)
          Snapshot.from_json(File.read(@path))
        rescue ex : JSON::ParseException
          raise "state decode failed: #{ex.message}"
        end
      end

      def save(snapshot : Snapshot)
        @lock.synchronize do
          dir = File.dirname(@path)
          Dir.mkdir_p(dir)
          tmp = "#{@path}.tmp"
          File.write(tmp, snapshot.to_pretty_json)
          File.rename(tmp, @path)
        end
      end
    end
  end
end
