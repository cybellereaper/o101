module O101
  module State
    struct Snapshot
      include JSON::Serializable
      getter version : String
      getter applied_files : Array(String)
      getter updated_at : String

      def initialize(@version = "", @applied_files = [] of String, @updated_at = Time.utc.to_rfc3339)
      end
    end

    class Store
      def initialize(@path : String)
      end

      def load : Snapshot
        return Snapshot.new if !File.exists?(@path)

        Snapshot.from_json(File.read(@path))
      rescue
        Snapshot.new
      end

      def save(snapshot : Snapshot)
        temp = "#{@path}.tmp"
        File.write(temp, snapshot.to_json)
        File.rename(temp, @path)
      ensure
        File.delete(temp) if temp && File.exists?(temp)
      end
    end
  end
end
