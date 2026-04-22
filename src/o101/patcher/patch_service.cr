module O101
  module Patcher
    class PatchService
      def initialize(@http : HttpClient = DefaultHttpClient.new)
      end

      def run(patch_info_url : String, install_dir : String, state_store : State::Store)
        patch_info = fetch_patch_info(patch_info_url)
        manifest = fetch_manifest(patch_info.manifest)

        snapshot = state_store.load
        applied = snapshot.applied_files.to_set

        manifest.files.each do |file|
          next if applied.includes?(file.dst)
          install_file(install_dir, patch_info.base_url, file)
          applied.add(file.dst)
        end

        state_store.save(State::Snapshot.new(version: manifest.version, applied_files: applied.to_a.sort, updated_at: Time.utc.to_rfc3339))
      end

      private def fetch_patch_info(url : String)
        payload = fetch_text(url)
        Manifest::Parser.parse_patch_info!(payload)
      end

      private def fetch_manifest(url : String)
        payload = fetch_text(url)
        Manifest::Parser.parse!(payload)
      end

      private def fetch_text(url : String) : String
        response = @http.get(url)
        raise Error.new("HTTP GET failed for #{url} with status #{response.status_code}") unless response.status_code == 200
        String.new(response.body)
      end

      private def install_file(install_dir : String, base_url : String, file : Manifest::FileEntry)
        response = @http.get(File.join(base_url, file.src))
        raise Error.new("download failed for #{file.src}") unless response.status_code == 200

        hash = Digest::SHA256.hexdigest(response.body)
        raise ValidationError.new("sha mismatch for #{file.dst}") unless hash == file.sha256.downcase
        raise ValidationError.new("size mismatch for #{file.dst}") unless response.body.size.to_i64 == file.size

        full_path = File.join(install_dir, file.dst)
        FileUtils.mkdir_p(File.dirname(full_path))
        File.write(full_path, response.body)
        File.chmod(full_path, file.mode)
      end
    end
  end
end
