import SwiftUI

struct ElapsedTimerText: View {
    let startedAt: Date
    var font: Font = .body

    var body: some View {
        Text(timerInterval: startedAt...(.distantFuture), countsDown: false, showsHours: true)
            .monospacedDigit()
            .font(font)
    }
}
