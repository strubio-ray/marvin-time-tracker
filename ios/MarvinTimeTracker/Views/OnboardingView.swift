import SwiftUI

struct OnboardingView: View {
    @Bindable var viewModel: TrackingViewModel

    @State private var serverURL = ""
    @State private var apiKey = ""
    @State private var isValidating = false
    @State private var validationError: String?

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Server URL", text: $serverURL)
                        .textContentType(.URL)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)

                    SecureField("API Key", text: $apiKey)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)
                } header: {
                    Text("Configuration")
                } footer: {
                    Text("Enter your relay server URL and API key.")
                }

                if let error = validationError {
                    Section {
                        Label(error, systemImage: "exclamationmark.triangle")
                            .foregroundStyle(.red)
                    }
                }

                Section {
                    Button {
                        Task { await validate() }
                    } label: {
                        HStack {
                            Text("Connect")
                            if isValidating {
                                Spacer()
                                ProgressView()
                            }
                        }
                    }
                    .disabled(serverURL.isEmpty || apiKey.isEmpty || isValidating)
                }
            }
            .navigationTitle("Setup")
        }
    }

    private func validate() async {
        isValidating = true
        validationError = nil
        defer { isValidating = false }

        let normalizedURL = serverURL.hasSuffix("/")
            ? String(serverURL.dropLast())
            : serverURL

        viewModel.saveCredentials(apiKey: apiKey, serverURL: normalizedURL)

        let isValid = await viewModel.validateServer()
        if !isValid {
            validationError = "Could not connect to server. Check URL and API key."
            viewModel.signOut()
        }
    }
}
