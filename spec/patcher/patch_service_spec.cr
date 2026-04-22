require "../spec_helper"

class FakeHttp
  include O101::Patcher::HttpClient

  def initialize(@responses : Hash(String, O101::Patcher::DownloadedFile))
  end

  def get(url : String) : O101::Patcher::DownloadedFile
    @responses[url]? || O101::Patcher::DownloadedFile.new(Bytes.empty, 404)
  end
end

describe O101::Patcher::PatchService do
  it "downloads, validates, and installs missing files" do
    install_dir = "/tmp/o101-install-#{Random.rand}"
    state_path = "/tmp/o101-state-#{Random.rand}.json"

    bytes = "ok".to_slice
    sha = Digest::SHA256.hexdigest(bytes)

    patch_info_url = "https://example/patch.json"
    manifest_url = "https://example/manifest.json"
    file_url = "https://cdn/files/Bin/app.bin"

    patch_info = %({"version":"1.0.0","manifest":"#{manifest_url}","base_url":"https://cdn"})
    manifest = %({"version":"1.0.0","files":[{"src":"files/Bin/app.bin","dst":"Bin/app.bin","size":2,"sha256":"#{sha}","mode":"0644"}]})

    http = FakeHttp.new({
      patch_info_url => O101::Patcher::DownloadedFile.new(patch_info.to_slice, 200),
      manifest_url   => O101::Patcher::DownloadedFile.new(manifest.to_slice, 200),
      file_url       => O101::Patcher::DownloadedFile.new(bytes, 200),
    })

    O101::Patcher::PatchService.new(http).run(patch_info_url, install_dir, O101::State::Store.new(state_path))

    File.read(File.join(install_dir, "Bin/app.bin")).should eq("ok")
  ensure
    FileUtils.rm_rf(install_dir.not_nil!) if install_dir
    File.delete(state_path.not_nil!) if state_path && File.exists?(state_path.not_nil!)
  end
end
