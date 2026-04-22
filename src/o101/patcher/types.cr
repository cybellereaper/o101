module O101
  module Patcher
    record DownloadedFile, body : Bytes, status_code : Int32

    module HttpClient
      abstract def get(url : String) : DownloadedFile
    end

    class DefaultHttpClient
      include HttpClient

      def get(url : String) : DownloadedFile
        response = HTTP::Client.get(url)
        DownloadedFile.new(response.body.to_slice, response.status_code)
      end
    end
  end
end
