require "http/client"

module Open101
  module Patcher
    class UpToDateError < Exception; end

    struct PatchInfo
      include JSON::Serializable
      getter version : String
      @[JSON::Field(key: "manifest")]
      getter manifest_url : String
      @[JSON::Field(key: "base_url")]
      getter base_url : String?
    end

    struct Config
      getter patch_info_url : String
      getter install_dir : String
      getter state_store : Open101::State::Store
      getter concurrency : Int32

      def initialize(@patch_info_url : String, @install_dir : String, @state_store : Open101::State::Store, @concurrency : Int32 = 4)
      end
    end

    class Runner
      def initialize(@config : Config)
      end

      def run
        info = fetch_patch_info
        manifest = Open101::Manifest::Parser.parse(fetch_text(info.manifest_url))
        snapshot = Open101::State::Snapshot.new(version: manifest.version)
        changed = false

        manifest.files.each do |entry|
          destination = File.join(@config.install_dir, entry.target.gsub('/', '/'))
          if valid_local_file?(destination, entry)
            snapshot.files[entry.target] = Open101::State::FileInfo.new(entry.size, entry.sha256.downcase)
            next
          end

          changed = true
          download_entry(info, entry, destination)
          snapshot.files[entry.target] = Open101::State::FileInfo.new(entry.size, entry.sha256.downcase)
        end

        @config.state_store.save(snapshot)
        raise UpToDateError.new("already up to date") unless changed
      end

      private def fetch_patch_info : PatchInfo
        PatchInfo.from_json(fetch_text(@config.patch_info_url))
      end

      private def fetch_text(url : String) : String
        response = HTTP::Client.get(url)
        raise "request failed for #{url}: #{response.status_code}" unless response.success?
        response.body
      end

      private def valid_local_file?(path : String, entry : Open101::Manifest::FileEntry) : Bool
        return false unless File.exists?(path)
        return false unless File.size(path) == entry.size
        digest = Digest::SHA256.hexdigest(File.read(path))
        digest == entry.sha256.downcase
      end

      private def download_entry(info : PatchInfo, entry : Open101::Manifest::FileEntry, destination : String)
        base = info.base_url || info.manifest_url
        source = URI.parse(base).resolve(entry.source).to_s
        response = HTTP::Client.get(source)
        raise "download failed: #{source} (#{response.status_code})" unless response.success?

        Dir.mkdir_p(File.dirname(destination))
        File.write(destination, response.body)

        if entry.mode
          File.chmod(destination, entry.mode.not_nil!.to_i(8))
        end

        raise "size mismatch for #{entry.target}" unless File.size(destination) == entry.size
        digest = Digest::SHA256.hexdigest(File.read(destination))
        raise "digest mismatch for #{entry.target}" unless digest == entry.sha256.downcase
      end
    end
  end
end
