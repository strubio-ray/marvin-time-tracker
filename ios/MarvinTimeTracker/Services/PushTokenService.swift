import Foundation

struct PushTokenService {
    private let serverURL: String
    private let apiKey: String

    init(serverURL: String, apiKey: String) {
        self.serverURL = serverURL
        self.apiKey = apiKey
    }

    func register(pushToStartToken: String? = nil, updateToken: String? = nil, deviceToken: String? = nil) async throws {
        let url = URL(string: "\(serverURL)/register")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")

        var body: [String: String] = [:]
        if let pushToStartToken {
            body["pushToStartToken"] = pushToStartToken
        }
        if let updateToken {
            body["updateToken"] = updateToken
        }
        if let deviceToken {
            body["deviceToken"] = deviceToken
        }

        request.httpBody = try JSONEncoder().encode(body)

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw URLError(.badServerResponse)
        }
    }
}
