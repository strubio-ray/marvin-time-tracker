import AppIntents
import Foundation

struct StopTrackingIntent: LiveActivityIntent {
    static let title: LocalizedStringResource = "Stop Tracking"
    static let description: IntentDescription = "Stops the current time tracking session"

    func perform() async throws -> some IntentResult {
        guard let defaults = UserDefaults(suiteName: "group.com.strubio.MarvinTimeTracker"),
              let serverURL = defaults.string(forKey: "serverURL") else {
            return .result()
        }

        let url = URL(string: "\(serverURL)/stop")!
        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = "{}".data(using: .utf8)

        _ = try? await URLSession.shared.data(for: request)

        return .result()
    }
}
