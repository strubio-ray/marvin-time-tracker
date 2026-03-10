import Foundation

protocol MarvinAPIClientProtocol: Sendable {
    func todayItems() async throws -> [MarvinTask]
    func startTracking(taskId: String, title: String) async throws
    func stopTracking(taskId: String?) async throws
    func fetchStatus() async throws -> ServerStatus
}

struct MarvinAPIClient: MarvinAPIClientProtocol {
    private let apiKey: String
    private let serverURL: String

    init(apiKey: String, serverURL: String) {
        self.apiKey = apiKey
        self.serverURL = serverURL
    }

    // MARK: - Server API calls

    func todayItems() async throws -> [MarvinTask] {
        let request = authorizedRequest(url: URL(string: "\(serverURL)/tasks")!, method: "GET")

        let (data, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            return []
        }

        return (try? JSONDecoder().decode([MarvinTask].self, from: data)) ?? []
    }

    func startTracking(taskId: String, title: String) async throws {
        var request = authorizedRequest(url: URL(string: "\(serverURL)/start")!, method: "POST")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(["taskId": taskId, "title": title])

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw APIError.serverError
        }
    }

    func stopTracking(taskId: String? = nil) async throws {
        var request = authorizedRequest(url: URL(string: "\(serverURL)/stop")!, method: "POST")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if let taskId {
            request.httpBody = try JSONEncoder().encode(["taskId": taskId])
        } else {
            request.httpBody = "{}".data(using: .utf8)
        }

        let (_, response) = try await URLSession.shared.data(for: request)
        guard let httpResponse = response as? HTTPURLResponse,
              httpResponse.statusCode == 200 else {
            throw APIError.serverError
        }
    }

    func fetchStatus() async throws -> ServerStatus {
        let request = authorizedRequest(url: URL(string: "\(serverURL)/status")!, method: "GET")
        let (data, _) = try await URLSession.shared.data(for: request)
        return try JSONDecoder().decode(ServerStatus.self, from: data)
    }

    // MARK: - Private

    private func authorizedRequest(url: URL, method: String) -> URLRequest {
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("Bearer \(apiKey)", forHTTPHeaderField: "Authorization")
        return request
    }

    enum APIError: Error {
        case serverError
    }
}

struct ServerStatus: Codable {
    let status: String
    let tracking: Bool
    let taskId: String?
    let taskTitle: String?
    let startedAt: Int64?
    let hasPushToStartToken: Bool
    let hasUpdateToken: Bool
}
