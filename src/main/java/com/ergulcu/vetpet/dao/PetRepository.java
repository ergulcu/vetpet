package com.ergulcu.vetpet.dao;

import com.ergulcu.vetpet.model.Pet;
import java.util.List;
import org.springframework.data.repository.CrudRepository;

/**
 *
 * @author Mehmet ERGÜLCÜ
 */
public interface PetRepository extends CrudRepository<Pet, Long> {
    List<Pet> findByNameContainingIgnoreCase(String name);
}
