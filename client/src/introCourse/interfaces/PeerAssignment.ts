export interface PeerInfo {
  courseParticipationId: string
  gitlabUsername: string
  seatName: string
  tutorGitlabUsername: string
}

export interface OwnPeerAssignment {
  peersIReview: PeerInfo[]
  peersWhoReviewMe: PeerInfo[]
}

export interface PeerAssignment {
  studentId: string
  peerId: string
}

export interface SyncResult {
  studentId: string
  peerId: string
  success: boolean
  error?: string
}
