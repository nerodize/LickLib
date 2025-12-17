package de.licklib.kotlinbackend.entity

import jakarta.persistence.Column
import jakarta.persistence.Entity
import jakarta.persistence.GeneratedValue
import jakarta.persistence.GenerationType
import jakarta.persistence.Id
import java.util.UUID

@Entity
class FileEntity(
    @Id @GeneratedValue(strategy = GenerationType.UUID)
    val id: UUID? = null,
    @Column(name = "name")
    val name: String,
    @Column(name = "size")
    val size: Long,
    @Column(name = "contentType")
    val contentType: String,
    @Column(name = "duration")
    val duration: Long,
)