require "../o101"

patch_info_url = ""
install_dir = ""
state_file = ""

OptionParser.parse do |parser|
  parser.banner = "Usage: wizturtle --patch-info URL --install-dir DIR --state-file FILE"
  parser.on("--patch-info URL", "Patch info URL") { |value| patch_info_url = value }
  parser.on("--install-dir DIR", "Installation directory") { |value| install_dir = value }
  parser.on("--state-file FILE", "State file path") { |value| state_file = value }
end

abort("--patch-info is required") if patch_info_url.blank?
abort("--install-dir is required") if install_dir.blank?
abort("--state-file is required") if state_file.blank?

store = O101::State::Store.new(state_file)
service = O101::Patcher::PatchService.new
service.run(patch_info_url, install_dir, store)
puts "patch complete"
