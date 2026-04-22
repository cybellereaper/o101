require "../o101"

login_addr = "127.0.0.1:12500"
game_addr = "127.0.0.1:12501"
zones = 50
zone_capacity = 10
max_players = 100

OptionParser.parse do |parser|
  parser.banner = "Usage: wizserver [options]"
  parser.on("--login-addr ADDR", "Login TCP addr") { |value| login_addr = value }
  parser.on("--game-addr ADDR", "Game TCP addr") { |value| game_addr = value }
  parser.on("--zones N", "Zone count") { |value| zones = value.to_i }
  parser.on("--zone-capacity N", "Per-zone capacity") { |value| zone_capacity = value.to_i }
  parser.on("--max-players N", "Max players") { |value| max_players = value.to_i }
end

realm = O101::WizServer::Realm.new(zones, zone_capacity, max_players)

[{"login", login_addr}, {"game", game_addr}].each do |(name, addr)|
  spawn do
    server = TCPServer.new(addr)
    puts "#{name} server listening on #{addr}"
    loop do
      socket = server.accept?
      next unless socket
      spawn do
        begin
          socket.puts "o101 #{name} ready"
          line = socket.gets
          socket.puts line if line
          if line
            id = "#{name}-#{Random.rand(1_000_000)}"
            realm.upsert_character(O101::WizServer::InGameCharacter.new(id, id, 1))
            zone = realm.assign_zone(id)
            socket.puts "zone=#{zone.id}"
          end
        ensure
          socket.close
        end
      end
    end
  end
end

sleep
