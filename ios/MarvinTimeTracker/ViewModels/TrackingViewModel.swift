import Foundation
import ActivityKit
import Observation

@MainActor
@Observable
final class TrackingViewModel {
    var trackingState: TrackingState = .idle
    var todayTasks: [MarvinTask] = []
    var isLoading = false
    var errorMessage: String?
    var showingTaskPicker = false

    var isOnboarded: Bool = KeychainService.isConfigured

    private let keychain: KeychainServiceProtocol
    private let apiClientFactory: (String, String) -> any MarvinAPIClientProtocol
    private let pushTokenServiceFactory: (String, String) -> any PushTokenServiceProtocol

    init(
        keychain: KeychainServiceProtocol = KeychainStore(),
        apiClientFactory: @escaping (String, String) -> any MarvinAPIClientProtocol = { MarvinAPIClient(apiKey: $0, serverURL: $1) },
        pushTokenServiceFactory: @escaping (String, String) -> any PushTokenServiceProtocol = { PushTokenService(serverURL: $0, apiKey: $1) }
    ) {
        self.keychain = keychain
        self.apiClientFactory = apiClientFactory
        self.pushTokenServiceFactory = pushTokenServiceFactory
    }

    private var apiClient: (any MarvinAPIClientProtocol)? {
        guard let apiKey = keychain.apiKey,
              let serverURL = keychain.serverURL else { return nil }
        return apiClientFactory(apiKey, serverURL)
    }

    private var pushTokenService: (any PushTokenServiceProtocol)? {
        guard let serverURL = keychain.serverURL,
              let apiKey = keychain.apiKey else { return nil }
        return pushTokenServiceFactory(serverURL, apiKey)
    }

    // MARK: - Onboarding

    func saveCredentials(apiKey: String, serverURL: String) {
        var kc = keychain
        kc.apiKey = apiKey
        kc.serverURL = serverURL
        isOnboarded = kc.isConfigured

        // Also save server URL to App Group for widget extension access
        let defaults = UserDefaults(suiteName: AppConstants.appGroupSuite)
        defaults?.set(serverURL, forKey: "serverURL")
    }

    func validateServer() async -> Bool {
        guard let client = apiClient else { return false }
        return (try? await client.fetchStatus()) != nil
    }

    // MARK: - Tracking

    func refreshStatus() async {
        guard let client = apiClient else { return }

        do {
            let status = try await client.fetchStatus()
            if status.tracking, let taskId = status.taskId, let title = status.taskTitle, let startedAtMs = status.startedAt {
                let startedAt = Date(timeIntervalSince1970: Double(startedAtMs) / 1000.0)
                trackingState = .tracking(taskId: taskId, title: title, startedAt: startedAt)

                // Start a Live Activity locally if none exists (e.g., push-to-start failed)
                if Activity<TimeTrackerAttributes>.activities.isEmpty {
                    await startLiveActivity(taskTitle: title, startedAt: startedAt)
                }
            } else {
                trackingState = .idle
            }
        } catch {
            errorMessage = "Failed to refresh status"
        }
    }

    func loadTodayTasks() async {
        guard let client = apiClient else { return }
        isLoading = true
        defer { isLoading = false }

        do {
            todayTasks = try await client.todayItems()
        } catch {
            errorMessage = "Failed to load tasks"
        }
    }

    func startTracking(task: MarvinTask) async {
        guard let client = apiClient else { return }

        do {
            try await client.startTracking(taskId: task.id, title: task.title)
            let startedAt = Date()
            trackingState = .tracking(taskId: task.id, title: task.title, startedAt: startedAt)
            showingTaskPicker = false

            // Start in-app Live Activity
            await startLiveActivity(taskTitle: task.title, startedAt: startedAt)
        } catch {
            errorMessage = "Failed to start tracking"
        }
    }

    func stopTracking() async {
        guard let client = apiClient else { return }

        do {
            try await client.stopTracking(taskId: trackingState.taskId)
            trackingState = .idle
        } catch {
            errorMessage = "Failed to stop tracking"
        }
    }

    // MARK: - Live Activity

    private func startLiveActivity(taskTitle: String, startedAt: Date) async {
        let attributes = TimeTrackerAttributes()
        let contentState = TimeTrackerAttributes.ContentState(
            taskTitle: taskTitle,
            startedAt: startedAt,
            isTracking: true
        )

        do {
            let activity = try Activity.request(
                attributes: attributes,
                content: .init(state: contentState, staleDate: nil),
                pushType: .token
            )

            // Observe update token for this activity
            Task {
                for await tokenData in activity.pushTokenUpdates {
                    let token = tokenData.map { String(format: "%02x", $0) }.joined()
                    do {
                        try await pushTokenService?.register(pushToStartToken: nil, updateToken: token, deviceToken: nil)
                    } catch {
                        print("Failed to register update token: \(error)")
                    }
                }
            }
        } catch {
            print("Failed to start Live Activity: \(error)")
        }
    }

    // MARK: - Push Tokens

    func observePushTokens() async {
        guard let service = pushTokenService else { return }

        for await tokenData in Activity<TimeTrackerAttributes>.pushToStartTokenUpdates {
            let token = tokenData.map { String(format: "%02x", $0) }.joined()
            do {
                try await service.register(pushToStartToken: token, updateToken: nil, deviceToken: nil)
            } catch {
                print("Failed to register push-to-start token: \(error)")
            }
        }
    }

    func observeActivityUpdates() async {
        for await activity in Activity<TimeTrackerAttributes>.activityUpdates {
            Task {
                for await tokenData in activity.pushTokenUpdates {
                    let token = tokenData.map { String(format: "%02x", $0) }.joined()
                    do {
                        try await pushTokenService?.register(pushToStartToken: nil, updateToken: token, deviceToken: nil)
                    } catch {
                        print("Failed to register activity update token: \(error)")
                    }
                }
            }
        }
    }

    func registerDeviceToken(_ token: String) async {
        do {
            try await pushTokenService?.register(pushToStartToken: nil, updateToken: nil, deviceToken: token)
        } catch {
            print("Failed to register device token: \(error)")
        }
    }

    func signOut() {
        var kc = keychain
        kc.apiKey = nil
        kc.serverURL = nil
        isOnboarded = false
        trackingState = .idle
        todayTasks = []
    }
}
