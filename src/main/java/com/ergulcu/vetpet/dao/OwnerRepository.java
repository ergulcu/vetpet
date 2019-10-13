package com.ergulcu.vetpet.dao;

import com.ergulcu.vetpet.model.Owner;
import java.util.List;
import org.springframework.data.repository.CrudRepository;

/**
 *
 * @author Mehmet ERGÜLCÜ
 */
public interface OwnerRepository extends CrudRepository<Owner, Long> {
    List<Owner> findByNameContainingIgnoreCase(String name);
}
