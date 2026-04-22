require "option_parser"
require "socket"
require "../open101"

game_dir = ""
login_addr = "127.0.0.1"
login_port = 12500

OptionParser.parse do |parser|
  parser.banner = "Usage: wizserver [options]"
  parser.on("--game-dir=DIR", "Game data directory") { |v| game_dir = v }
  parser.on("--login-addr=ADDR", "Login bind address") do |v|
    host, port = v.split(":", 2)
    login_addr = host
    login_port = port.to_i
  end
end

if game_dir.empty? || !Dir.exists?(game_dir)
  STDERR.puts "--game-dir must be a valid directory"
  exit 2
end

server = TCPServer.new(login_addr, login_port)
puts "wizserver listening on #{login_addr}:#{login_port}"
loop do
  client = server.accept?
  next unless client
  spawn { Open101::WizServer::Util.handle_connection(client, "login") }
end
