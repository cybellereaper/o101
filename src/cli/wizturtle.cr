require "option_parser"
require "../open101"

patch_info = ""
install_dir = "."
state_file = ""

OptionParser.parse do |parser|
  parser.banner = "Usage: wizturtle [options]"
  parser.on("--patch-info=URL", "Patch info URL") { |v| patch_info = v }
  parser.on("--install-dir=DIR", "Install directory") { |v| install_dir = v }
  parser.on("--state-file=FILE", "State file path") { |v| state_file = v }
end

if patch_info.empty?
  STDERR.puts "--patch-info is required"
  exit 2
end

state_file = File.join(install_dir, ".wizturtle", "state.json") if state_file.empty?
store = Open101::State::Store.new(state_file)
config = Open101::Patcher::Config.new(patch_info, install_dir, store)
runner = Open101::Patcher::Runner.new(config)

begin
  runner.run
  puts "Patch completed successfully"
rescue ex : Open101::Patcher::UpToDateError
  puts ex.message
end
